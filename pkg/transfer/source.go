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

// ImageSource is a reference to a remote image need to be pulled.
type ImageSource struct {
	registry   string
	repository string
	tag        string
	sourceRef  types.ImageReference
	source     types.ImageSource
	ctx        context.Context
	sysctx     *types.SystemContext
}

// NewImageSource generates a PullJob by repository, the repository string must include "tag",
// if username or password is empty, access to repository will be anonymous.
// a repository string is the rest part of the images url except "tag" and "registry"
func NewImageSource(registry, repository, tag, username, password string, insecure bool) (*ImageSource, error) {
	if utils.CheckIfIncludeTag(repository) {
		return nil, fmt.Errorf("repository string should not include tag")
	}

	// tag may be empty
	tagWithColon := ""
	if tag != "" {
		tagWithColon = ":" + tag
	}

	srcRef, err := docker.ParseReference("//" + registry + "/" + repository + tagWithColon)
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

	ctx := context.WithValue(context.Background(), interface{}("ImageSource"), repository)
	if username != "" && password != "" {
		sysctx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: username,
			Password: password,
		}
	}

	var rawSource types.ImageSource
	if tag != "" {
		// if tag is empty, will attach to the "latest" tag, and will get a error if "latest" is not exist
		rawSource, err = srcRef.NewImageSource(ctx, sysctx)
		if err != nil {
			return nil, err
		}
	}

	return &ImageSource{
		sourceRef:  srcRef,
		source:     rawSource,
		ctx:        ctx,
		sysctx:     sysctx,
		registry:   registry,
		repository: repository,
		tag:        tag,
	}, nil
}

// GetManifest get manifest file from source image
func (i *ImageSource) GetManifest() ([]byte, string, error) {
	if i.source == nil {
		return nil, "", fmt.Errorf("can not get manifest file without specfied a tag")
	}
	return i.source.GetManifest(i.ctx, nil)
}

// GetBlobInfos get blobs from source image.
func (i *ImageSource) GetBlobInfos(manifestByte []byte, manifestType string) ([]types.BlobInfo, error) {
	if i.source == nil {
		return nil, fmt.Errorf("can not get blobs without specfied a tag")
	}

	manifestInfoSlice, err := ManifestHandler(manifestByte, manifestType, i)
	if err != nil {
		return nil, err
	}

	// get a manifest

	srcBlobs := []types.BlobInfo{}

	for _, manifestInfo := range manifestInfoSlice {
		blobInfos := manifestInfo.LayerInfos()
		for _, l := range blobInfos {
			srcBlobs = append(srcBlobs, l.BlobInfo)
		}
		// append config blob info
		configBlob := manifestInfo.ConfigInfo()
		if configBlob.Digest != "" {
			srcBlobs = append(srcBlobs, configBlob)
		}
	}

	return srcBlobs, nil
}

// GetABlob gets a blob from remote image
func (i *ImageSource) GetABlob(blobInfo types.BlobInfo) (io.ReadCloser, int64, error) {
	return i.source.GetBlob(i.ctx, types.BlobInfo{Digest: blobInfo.Digest, Size: -1}, NoCache)
}

// Close an ImageSource
func (i *ImageSource) Close() error {
	return i.source.Close()
}

// GetRegistry returns the registry of a ImageSource
func (i *ImageSource) GetRegistry() string {
	return i.registry
}

// GetRepository returns the repository of a ImageSource
func (i *ImageSource) GetRepository() string {
	return i.repository
}

// GetTag returns the tag of a ImageSource
func (i *ImageSource) GetTag() string {
	return i.tag
}

// GetSourceRepoTags gets all the tags of a repository which ImageSource belongs to
func (i *ImageSource) GetSourceRepoTags() ([]string, error) {
	return docker.GetRepositoryTags(i.ctx, i.sysctx, i.sourceRef)
}
