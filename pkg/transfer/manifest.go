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
	"fmt"

	"github.com/containers/image/v5/manifest"
)

// ManifestSchemaV2List describes a schema V2 manifest list
type ManifestSchemaV2List struct {
	Manifests []ManifestSchemaV2Info `json:"manifests"`
}

// ManifestSchemaV2Info includes of the imformation needes of a schema V2 manifest file
type ManifestSchemaV2Info struct {
	Digest string `json:"digest"`
}

// ManifestHandler expends the ability of handling manifest list in schema2, but it's not finished yet
// return the digest array of manifests in the manifest list if exist.
func ManifestHandler(m []byte, t string, i *ImageSource) ([]manifest.Manifest, error) {
	var manifestInfoSlice []manifest.Manifest

	if t == manifest.DockerV2Schema2MediaType {
		manifestInfo, err := manifest.Schema2FromManifest(m)
		if err != nil {
			return nil, err
		}
		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, nil
	} else if t == manifest.DockerV2Schema1MediaType || t == manifest.DockerV2Schema1SignedMediaType {
		manifestInfo, err := manifest.Schema1FromManifest(m)
		if err != nil {
			return nil, err
		}
		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, nil
	} else if t == manifest.DockerV2ListMediaType {

		manifestSchemaListInfo, err := manifest.Schema2ListFromManifest(m)
		if err != nil {
			return nil, err
		}

		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {

			manifestByte, manifestType, err := i.source.GetManifest(i.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return nil, err
			}

			platformSpecManifest, err := ManifestHandler(manifestByte, manifestType, i)
			if err != nil {
				return nil, err
			}

			manifestInfoSlice = append(manifestInfoSlice, platformSpecManifest...)
		}
		return manifestInfoSlice, nil
	}
	return nil, fmt.Errorf("unsupported manifest type: %v", t)
}
