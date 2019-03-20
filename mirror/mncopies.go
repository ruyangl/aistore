// Package mirror provides local mirroring and replica management
/*
 * Copyright (c) 2018, NVIDIA CORPORATION. All rights reserved.
 */
package mirror

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/NVIDIA/aistore/3rdparty/glog"
	"github.com/NVIDIA/aistore/cluster"
	"github.com/NVIDIA/aistore/cmn"
	"github.com/NVIDIA/aistore/fs"
)

const (
	throttleNumObjects = 16                      // unit of self-throttling
	logNumProcessed    = throttleNumObjects * 16 // unit of house-keeping
	//
	MaxNCopies = 8
)

// XactBckMakeNCopies (extended action) reduces data redundancy of a given bucket to 1 (single copy)
// It runs in a background and traverses all local mountpaths to do the job.

type (
	XactBckMakeNCopies struct {
		// implements cmn.Xact a cmn.Runner interfaces
		cmn.XactBase
		// runtime
		doneCh  chan struct{}
		joggers map[string]*jogger
		// init
		T          cluster.Target
		Namelocker cluster.NameLocker
		Copies     int
		BckIsLocal bool
	}
	jogger struct { // one per mountpath
		parent    *XactBckMakeNCopies
		mpathInfo *fs.MountpathInfo
		config    *cmn.Config
		num       int64
		stopCh    chan struct{}
	}
)

//
// public methods
//

func (r *XactBckMakeNCopies) Run() (err error) {
	var numjs int
	if numjs, err = r.init(); err != nil {
		return err
	}
	glog.Infoln(r.String())
	// control loop
	for {
		select {
		case <-r.ChanAbort():
			r.stop()
			return fmt.Errorf("%s aborted, exiting", r)
		case <-r.doneCh:
			numjs--
			if numjs == 0 {
				glog.Infof("%s: all done", r)
				r.joggers = nil
				r.stop()
				return
			}
		}
	}
}

func (r *XactBckMakeNCopies) Stop(error) { r.Abort() } // call base method

func ValidateNCopies(copies int) error {
	if copies <= 0 || copies > MaxNCopies {
		return fmt.Errorf("Invalid num copies %d (valid range 1 - %d)", copies, MaxNCopies)
	}
	return nil
}

//
// private methods
//

func (r *XactBckMakeNCopies) init() (numjs int, err error) {
	if err = ValidateNCopies(r.Copies); err != nil {
		return
	}
	availablePaths, _ := fs.Mountpaths.Get()
	numjs = len(availablePaths)
	if err = checkErrNumMp(r, numjs); err != nil {
		return
	}
	r.doneCh = make(chan struct{}, numjs)
	r.joggers = make(map[string]*jogger, numjs)
	config := cmn.GCO.Get()
	for _, mpathInfo := range availablePaths {
		jogger := &jogger{parent: r, mpathInfo: mpathInfo, config: config}
		mpathLC := mpathInfo.MakePath(fs.ObjectType, r.BckIsLocal)
		r.joggers[mpathLC] = jogger
		go jogger.jog()
	}
	return
}

func (r *XactBckMakeNCopies) stop() {
	if r.Finished() {
		glog.Warningf("%s is (already) not running", r)
		return
	}
	for _, jogger := range r.joggers {
		jogger.stop()
	}
	r.EndTime(time.Now())
}

//
// mpath jogger
//
func (j *jogger) stop() { j.stopCh <- struct{}{}; close(j.stopCh) }

func (j *jogger) jog() {
	glog.Infof("jogger[%s/%s] started", j.mpathInfo, j.parent.Bucket())
	j.stopCh = make(chan struct{}, 1)
	dir := j.mpathInfo.MakePathBucket(fs.ObjectType, j.parent.Bucket(), j.parent.BckIsLocal)
	if err := filepath.Walk(dir, j.walk); err != nil {
		s := err.Error()
		if strings.Contains(s, "xaction") {
			glog.Infof("%s: stopping traversal: %s", dir, s)
		} else {
			glog.Errorf("%s: failed to traverse, err: %v", dir, err)
		}
	}
	j.parent.doneCh <- struct{}{}
}

func (j *jogger) walk(fqn string, osfi os.FileInfo, err error) error {
	if err != nil {
		if errstr := cmn.PathWalkErr(err); errstr != "" {
			glog.Errorf(errstr)
			return err
		}
		return nil
	}
	if osfi.Mode().IsDir() {
		return nil
	}
	lom := &cluster.LOM{T: j.parent.T, FQN: fqn}
	if errstr := lom.Fill("", cluster.LomFstat|cluster.LomCopy, j.config); errstr != "" || !lom.Exists() {
		if glog.V(4) {
			glog.Infof("Warning: %s", errstr)
		}
		return nil
	}
	cmn.Assert(j.parent.BckIsLocal == lom.BckIsLocal)
	if n := lom.NumCopies(); n == j.parent.Copies {
		return nil
	} else if n > j.parent.Copies {
		err = j.delCopies(lom)
	} else {
		err = j.addCopies(lom)
	}
	j.num++
	if (j.num % throttleNumObjects) == 0 {
		if err = j.yieldTerm(); err != nil {
			return err
		}
		if (j.num % logNumProcessed) == 0 {
			glog.Infof("jogger[%s/%s] erased %d copies...", j.mpathInfo, j.parent.Bucket(), j.num)
			j.config = cmn.GCO.Get()
		}
	} else {
		runtime.Gosched()
	}
	return err
}

func (j *jogger) delCopies(lom *cluster.LOM) (err error) {
	j.parent.Namelocker.Lock(lom.Uname, true)
	if j.parent.Copies == 1 {
		if errstr := lom.DelAllCopies(); errstr != "" {
			err = errors.New(errstr)
		}
	} else {
		for i := len(lom.CopyFQN) - 1; i >= j.parent.Copies-1; i-- {
			cpyfqn := lom.CopyFQN[i]
			if errstr := lom.DelCopy(cpyfqn); errstr != "" {
				err = errors.New(errstr)
				break
			}
		}
	}
	j.parent.Namelocker.Unlock(lom.Uname, true)
	return
}

func (j *jogger) addCopies(lom *cluster.LOM) error {
	return nil
}

// [throttle]
func (j *jogger) yieldTerm() error {
	xaction := &j.config.Xaction
	select {
	case <-j.stopCh:
		return fmt.Errorf("jogger[%s/%s] aborted, exiting", j.mpathInfo, j.parent.Bucket())
	default:
		_, curr := j.mpathInfo.GetIOstats(fs.StatDiskUtil)
		if curr.Max >= float32(xaction.DiskUtilHighWM) && curr.Min > float32(xaction.DiskUtilLowWM) {
			time.Sleep(cmn.ThrottleSleepAvg)
		} else {
			time.Sleep(cmn.ThrottleSleepMin)
		}
		break
	}
	return nil
}

// common helper
func checkErrNumMp(xx cmn.Xact, l int) error {
	if l < 2 {
		return fmt.Errorf("%s: number of mountpaths (%d) is insufficient for local mirroring, exiting", xx, l)
	}
	return nil
}
