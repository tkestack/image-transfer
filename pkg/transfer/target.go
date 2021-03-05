/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2020 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package transfer

import (
	"context"
	"fmt"
	"io"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	"tkestack.io/image-transfer/pkg/utils"
)

// ImageTarget is a reference of a remote image we will push to
type ImageTarget struct {
	registry   string
	repository string
	tag        string
	targetRef  types.ImageReference
	target     types.ImageDestination
	ctx        context.Context
}

// NewImageTarget generates a ImageTarget by repository, the repository string must include "tag".
// If username or password is empty, access to repository will be anonymous.
func NewImageTarget(registry, repository, tag, username, password string, insecure bool) (*ImageTarget, error) {
	if utils.CheckIfIncludeTag(repository) {
		return nil, fmt.Errorf("repository string should not include tag")
	}

	// tag may be empty
	tagWithColon := ""
	if tag != "" {
		tagWithColon = ":" + tag
	}

	// if tag is empty, will attach to the "latest" tag
	destRef, err := docker.ParseReference("//" + registry + "/" + repository + tagWithColon)
	if err != nil {
		return nil, err
	}

	var sysctx *types.SystemContext
	if insecure {
		// destinatoin registry is http service
		sysctx = &types.SystemContext{
			DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
		}
	} else {
		sysctx = &types.SystemContext{}
	}

	ctx := context.WithValue(context.Background(), interface{}("ImageTarget"), repository)
	if username != "" && password != "" {
		sysctx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: username,
			Password: password,
		}
	}

	rawtarget, err := destRef.NewImageDestination(ctx, sysctx)
	if err != nil {
		return nil, err
	}

	return &ImageTarget{
		targetRef: destRef,
		target:    rawtarget,
		ctx:       ctx,

		registry:   registry,
		repository: repository,
		tag:        tag,
	}, nil
}

// PushManifest push a manifest file to target image
func (i *ImageTarget) PushManifest(manifestByte []byte) error {
	return i.target.PutManifest(i.ctx, manifestByte, nil)
}

// PutABlob push a blob to target image
func (i *ImageTarget) PutABlob(blob io.ReadCloser, blobInfo types.BlobInfo) error {
	_, err := i.target.PutBlob(i.ctx, blob, types.BlobInfo{
		Digest: blobInfo.Digest,
		Size:   blobInfo.Size,
	}, NoCache, true)

	// io.ReadCloser need to be close
	defer blob.Close()

	return err
}

// CheckBlobExist checks if a blob exist for target and reuse exist blobs
func (i *ImageTarget) CheckBlobExist(blobInfo types.BlobInfo) (bool, error) {
	exist, _, err := i.target.TryReusingBlob(i.ctx, types.BlobInfo{
		Digest: blobInfo.Digest,
		Size:   blobInfo.Size,
	}, NoCache, false)

	return exist, err
}

// Close a ImageTarget
func (i *ImageTarget) Close() error {
	return i.target.Close()
}

// GetRegistry returns the registry of a ImageTarget
func (i *ImageTarget) GetRegistry() string {
	return i.registry
}

// GetRepository returns the repository of a ImageTarget
func (i *ImageTarget) GetRepository() string {
	return i.repository
}

// GetTag return the tag of a ImageTarget
func (i *ImageTarget) GetTag() string {
	return i.tag
}
