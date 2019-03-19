// Package api provides RESTful API to AIS object storage
/*
 * Copyright (c) 2018, NVIDIA CORPORATION. All rights reserved.
 */
package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/NVIDIA/aistore/cmn"
	jsoniter "github.com/json-iterator/go"
)

// SetBucketProps API
//
// Set the properties of a bucket, using the bucket name and the bucket properties to be set.
// Validation of the properties passed in is performed by AIStore Proxy.
func SetBucketProps(baseParams *BaseParams, bucket string, props cmn.BucketProps, query ...url.Values) error {
	if props.Cksum.Type == "" {
		props.Cksum.Type = cmn.ChecksumInherit
	}

	b, err := jsoniter.Marshal(cmn.ActionMsg{Action: cmn.ActSetProps, Value: props})
	if err != nil {
		return err
	}

	optParams := OptionalParams{}
	if len(query) > 0 {
		optParams.Query = query[0]
	}
	baseParams.Method = http.MethodPut
	path := cmn.URLPath(cmn.Version, cmn.Buckets, bucket)
	_, err = DoHTTPRequest(baseParams, path, b, optParams)
	return err
}

// SetBucketProp API
//
// Set a single propertie of a bucket, using the bucket name, and the bucket property's name and value to be set.
// The function converts a property to a string to simplify route selection(it is
// either single string value or entire bucket property structure) and to avoid
// a lot of type reflexlection checks.
// Validation of the properties passed in is performed by AIStore Proxy.
func SetBucketProp(baseParams *BaseParams, bucket, prop string, value interface{}, query ...url.Values) error {
	strValue := fmt.Sprintf("%v", value)
	b, err := jsoniter.Marshal(cmn.ActionMsg{Action: cmn.ActSetProps, Name: prop, Value: strValue})
	if err != nil {
		return err
	}
	optParams := OptionalParams{}
	if len(query) > 0 {
		optParams.Query = query[0]
	}
	baseParams.Method = http.MethodPut
	path := cmn.URLPath(cmn.Version, cmn.Buckets, bucket)
	_, err = DoHTTPRequest(baseParams, path, b, optParams)
	return err
}

// ResetBucketProps API
//
// Reset the properties of a bucket, identified by its name, to the global configuration.
func ResetBucketProps(baseParams *BaseParams, bucket string, query ...url.Values) error {
	b, err := jsoniter.Marshal(cmn.ActionMsg{Action: cmn.ActResetProps})
	if err != nil {
		return err
	}
	optParams := OptionalParams{}
	if len(query) > 0 {
		optParams.Query = query[0]
	}
	baseParams.Method = http.MethodPut
	path := cmn.URLPath(cmn.Version, cmn.Buckets, bucket)
	_, err = DoHTTPRequest(baseParams, path, b, optParams)
	return err
}

// HeadBucket API
//
// Returns the properties of a bucket specified by its name.
// Converts the string type fields returned from the HEAD request to their
// corresponding counterparts in the BucketProps struct
func HeadBucket(baseParams *BaseParams, bucket string, query ...url.Values) (*cmn.BucketProps, error) {
	baseParams.Method = http.MethodHead
	path := cmn.URLPath(cmn.Version, cmn.Buckets, bucket)
	optParams := OptionalParams{}
	if len(query) > 0 {
		optParams.Query = query[0]
	}

	r, err := doHTTPRequestGetResp(baseParams, path, nil, optParams)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	cksumProps := cmn.CksumConf{
		Type: r.Header.Get(cmn.HeaderBucketChecksumType),
	}
	if b, err := strconv.ParseBool(r.Header.Get(cmn.HeaderBucketValidateColdGet)); err == nil {
		cksumProps.ValidateColdGet = b
	}
	if b, err := strconv.ParseBool(r.Header.Get(cmn.HeaderBucketValidateWarmGet)); err == nil {
		cksumProps.ValidateWarmGet = b
	}
	if b, err := strconv.ParseBool(r.Header.Get(cmn.HeaderBucketValidateRange)); err == nil {
		cksumProps.EnableReadRange = b
	}

	lruProps := cmn.LRUConf{
		DontEvictTimeStr:   r.Header.Get(cmn.HeaderBucketDontEvictTime),
		CapacityUpdTimeStr: r.Header.Get(cmn.HeaderBucketCapUpdTime),
	}
	if b, err := strconv.ParseUint(r.Header.Get(cmn.HeaderBucketLRULowWM), 10, 32); err == nil {
		lruProps.LowWM = int64(b)
	}
	if b, err := strconv.ParseUint(r.Header.Get(cmn.HeaderBucketLRUHighWM), 10, 32); err == nil {
		lruProps.HighWM = int64(b)
	}
	if b, err := strconv.ParseInt(r.Header.Get(cmn.HeaderBucketAtimeCacheMax), 10, 32); err == nil {
		lruProps.AtimeCacheMax = b
	}
	if b, err := strconv.ParseBool(r.Header.Get(cmn.HeaderBucketLRUEnabled)); err == nil {
		lruProps.Enabled = b
	}

	mirrorProps := cmn.MirrorConf{}
	if b, err := strconv.ParseInt(r.Header.Get(cmn.HeaderBucketCopies), 10, 32); err == nil {
		mirrorProps.Copies = b
	}
	if b, err := strconv.ParseBool(r.Header.Get(cmn.HeaderBucketMirrorEnabled)); err == nil {
		mirrorProps.Enabled = b
	}
	if n, err := strconv.ParseInt(r.Header.Get(cmn.HeaderBucketMirrorThresh), 10, 32); err == nil {
		mirrorProps.UtilThresh = n
	}

	ecProps := cmn.ECConf{}
	if b, err := strconv.ParseBool(r.Header.Get(cmn.HeaderBucketECEnabled)); err == nil {
		ecProps.Enabled = b
	}
	if n, err := strconv.ParseInt(r.Header.Get(cmn.HeaderBucketECMinSize), 10, 64); err == nil {
		ecProps.ObjSizeLimit = n
	}
	if n, err := strconv.ParseInt(r.Header.Get(cmn.HeaderBucketECData), 10, 32); err == nil {
		ecProps.DataSlices = int(n)
	}
	if n, err := strconv.ParseInt(r.Header.Get(cmn.HeaderBucketECParity), 10, 32); err == nil {
		ecProps.ParitySlices = int(n)
	}

	return &cmn.BucketProps{
		CloudProvider: r.Header.Get(cmn.HeaderCloudProvider),
		Versioning:    r.Header.Get(cmn.HeaderVersioning),
		NextTierURL:   r.Header.Get(cmn.HeaderNextTierURL),
		ReadPolicy:    r.Header.Get(cmn.HeaderReadPolicy),
		WritePolicy:   r.Header.Get(cmn.HeaderWritePolicy),
		Cksum:         cksumProps,
		LRU:           lruProps,
		Mirror:        mirrorProps,
		EC:            ecProps,
	}, nil
}

// GetBucketNames API
//
// bckProvider takes one of "" (empty), "cloud" or "local". If bckProvider is empty, return all bucketnames.
// Otherwise return "cloud" or "local" buckets.
func GetBucketNames(baseParams *BaseParams, bckProvider string) (*cmn.BucketNames, error) {
	bucketNames := &cmn.BucketNames{}
	baseParams.Method = http.MethodGet
	path := cmn.URLPath(cmn.Version, cmn.Buckets, cmn.ListAll)
	query := url.Values{cmn.URLParamBckProvider: []string{bckProvider}}
	optParams := OptionalParams{Query: query}

	b, err := DoHTTPRequest(baseParams, path, nil, optParams)
	if err != nil {
		return nil, err
	}
	if len(b) != 0 {
		err = jsoniter.Unmarshal(b, &bucketNames)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal bucket names, err: %v - [%s]", err, string(b))
		}
	} else {
		return nil, fmt.Errorf("empty response instead of empty bucket list from %s", baseParams.URL)
	}
	return bucketNames, nil
}

// CreateLocalBucket API
//
// CreateLocalBucket sends a HTTP request to a proxy to create a local bucket with the given name
func CreateLocalBucket(baseParams *BaseParams, bucket string) error {
	msg, err := jsoniter.Marshal(cmn.ActionMsg{Action: cmn.ActCreateLB})
	if err != nil {
		return err
	}
	baseParams.Method = http.MethodPost
	path := cmn.URLPath(cmn.Version, cmn.Buckets, bucket)
	_, err = DoHTTPRequest(baseParams, path, msg)
	return err
}

// DestroyLocalBucket API
//
// DestroyLocalBucket sends a HTTP request to a proxy to remove a local bucket with the given name
func DestroyLocalBucket(baseParams *BaseParams, bucket string) error {
	b, err := jsoniter.Marshal(cmn.ActionMsg{Action: cmn.ActDestroyLB})
	if err != nil {
		return err
	}
	baseParams.Method = http.MethodDelete
	path := cmn.URLPath(cmn.Version, cmn.Buckets, bucket)
	_, err = DoHTTPRequest(baseParams, path, b)
	return err
}

// EvictCloudBucket API
//
// EvictCloudBucket sends a HTTP request to a proxy to evict a cloud bucket with the given name
func EvictCloudBucket(baseParams *BaseParams, bucket string, query ...url.Values) error {
	b, err := jsoniter.Marshal(cmn.ActionMsg{Action: cmn.ActEvictCB})
	if err != nil {
		return err
	}
	optParams := OptionalParams{}
	if len(query) > 0 {
		optParams.Query = query[0]
	}
	baseParams.Method = http.MethodDelete
	path := cmn.URLPath(cmn.Version, cmn.Buckets, bucket)
	_, err = DoHTTPRequest(baseParams, path, b, optParams)
	return err
}

// RenameLocalBucket API
//
// RenameLocalBucket changes the name of a bucket from oldName to newBucketName
func RenameLocalBucket(baseParams *BaseParams, oldName, newName string) error {
	b, err := jsoniter.Marshal(cmn.ActionMsg{Action: cmn.ActRenameLB, Name: newName})
	if err != nil {
		return err
	}
	baseParams.Method = http.MethodPost
	path := cmn.URLPath(cmn.Version, cmn.Buckets, oldName)
	_, err = DoHTTPRequest(baseParams, path, b)
	return err
}

// ListBucket API
//
// ListBucket returns list of objects in a bucket. numObjects is the
// maximum number of objects returned by ListBucket (0 - return all objects in a bucket)
func ListBucket(baseParams *BaseParams, bucket string, msg *cmn.GetMsg, numObjects int, query ...url.Values) (*cmn.BucketList, error) {
	baseParams.Method = http.MethodPost
	path := cmn.URLPath(cmn.Version, cmn.Buckets, bucket)
	reslist := &cmn.BucketList{Entries: make([]*cmn.BucketEntry, 0, 1000)}
	q := url.Values{}
	if len(query) > 0 {
		q = query[0]
	}

	// An optimization to read as few objects from bucket as possible.
	// toRead is the current number of objects ListBucket must read before
	// returning the list. Every cycle the loop reads objects by pages and
	// decreases toRead by the number of received objects. When toRead gets less
	// than pageSize, the loop does the final request with reduced pageSize
	toRead := numObjects
	for {
		if toRead != 0 {
			if (msg.GetPageSize == 0 && toRead < cmn.DefaultPageSize) ||
				(msg.GetPageSize != 0 && msg.GetPageSize > toRead) {
				msg.GetPageSize = toRead
			}
		}

		b, err := jsoniter.Marshal(cmn.ActionMsg{Action: cmn.ActListObjects, Value: msg})
		if err != nil {
			return nil, err
		}

		optParams := OptionalParams{Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
			Query: q}
		respBody, err := DoHTTPRequest(baseParams, path, b, optParams)
		if err != nil {
			return nil, err
		}

		page := &cmn.BucketList{}
		page.Entries = make([]*cmn.BucketEntry, 0, 1000)

		if err = jsoniter.Unmarshal(respBody, page); err != nil {
			return nil, fmt.Errorf("failed to json-unmarshal, err: %v [%s]", err, string(b))
		}

		reslist.Entries = append(reslist.Entries, page.Entries...)
		if page.PageMarker == "" {
			msg.GetPageMarker = ""
			break
		}

		if numObjects != 0 {
			if len(reslist.Entries) >= numObjects {
				break
			}
			toRead -= len(page.Entries)
		}

		msg.GetPageMarker = page.PageMarker
	}

	return reslist, nil
}

// ListBuckeFast returns list of objects in a bucket.
// Build an object list with minimal set of properties: name and size.
// All GetMsg fields except prefix do not work and are skipped.
// Function always returns the whole list of objects without paging
func ListBucketFast(baseParams *BaseParams, bucket string, msg *cmn.GetMsg, query ...url.Values) (*cmn.BucketList, error) {
	var (
		b   []byte
		err error
	)
	baseParams.Method = http.MethodPost
	path := cmn.URLPath(cmn.Version, cmn.Buckets, bucket, cmn.ActListObjects)
	reslist := &cmn.BucketList{}
	if msg != nil {
		b, err = jsoniter.Marshal(cmn.ActionMsg{Action: cmn.ActListObjects, Value: msg})
		if err != nil {
			return nil, err
		}
	}
	q := url.Values{}
	if len(query) > 0 {
		q = query[0]
	}

	optParams := OptionalParams{Header: http.Header{
		"Content-Type": []string{"application/json"},
	},
		Query: q}
	respBody, err := DoHTTPRequest(baseParams, path, b, optParams)
	if err != nil {
		return nil, err
	}

	if err = jsoniter.Unmarshal(respBody, reslist); err != nil {
		return nil, fmt.Errorf("failed to json-unmarshal, err: %v [%s]", err, string(b))
	}

	return reslist, nil
}

// MakeNCopies API
//
// MakeNCopies starts an extended action (xaction) to bring a given bucket to a certain redundancy level (num copies)
func MakeNCopies(baseParams *BaseParams, bucket string, copies int) error {
	b, err := jsoniter.Marshal(cmn.ActionMsg{Action: cmn.ActMakeNCopies, Value: copies})
	if err != nil {
		return err
	}
	baseParams.Method = http.MethodPost
	path := cmn.URLPath(cmn.Version, cmn.Buckets, bucket)
	_, err = DoHTTPRequest(baseParams, path, b)
	return err
}
