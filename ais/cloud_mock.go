// Package ais provides core functionality for the AIStore object storage.
/*
 * Copyright (c) 2018, NVIDIA CORPORATION. All rights reserved.
 */
package ais

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/NVIDIA/aistore/cluster"
	"github.com/NVIDIA/aistore/cmn"
)

type (
	emptyCloud struct{} // mock
)

const bucketDoesNotExist = "bucket %q %s"

var (
	_ cloudif = &emptyCloud{}
)

func newEmptyCloud() *emptyCloud { return &emptyCloud{} }

func (m *emptyCloud) listbucket(ctx context.Context, bucket string, msg *cmn.GetMsg) (jsbytes []byte, errstr string, errcode int) {
	return []byte{}, fmt.Sprintf(bucketDoesNotExist, bucket, cmn.DoesNotExist), http.StatusNotFound
}
func (m *emptyCloud) headbucket(ctx context.Context, bucket string) (bucketprops cmn.SimpleKVs, errstr string, errcode int) {
	return cmn.SimpleKVs{}, fmt.Sprintf(bucketDoesNotExist, bucket, cmn.DoesNotExist), http.StatusNotFound
}
func (m *emptyCloud) headobject(ctx context.Context, bucket string, objname string) (objmeta cmn.SimpleKVs, errstr string, errcode int) {
	return cmn.SimpleKVs{}, fmt.Sprintf(bucketDoesNotExist, bucket, cmn.DoesNotExist), http.StatusNotFound
}
func (m *emptyCloud) getobj(ctx context.Context, fqn, bucket, objname string) (props *cluster.LOM, errstr string, errcode int) {
	return nil, fmt.Sprintf(bucketDoesNotExist, bucket, cmn.DoesNotExist), http.StatusNotFound
}
func (m *emptyCloud) putobj(ctx context.Context, file *os.File, bucket, objname string, cksum cmn.CksumProvider) (version string, errstr string, errcode int) {
	return "", fmt.Sprintf(bucketDoesNotExist, bucket, cmn.DoesNotExist), http.StatusNotFound
}
func (m *emptyCloud) deleteobj(ctx context.Context, bucket, objname string) (errstr string, errcode int) {
	return fmt.Sprintf(bucketDoesNotExist, bucket, cmn.DoesNotExist), http.StatusNotFound
}

// the function must not fail - it should return empty list
func (m *emptyCloud) getbucketnames(ctx context.Context) (buckets []string, errstr string, errcode int) {
	return []string{}, "", 0
}
