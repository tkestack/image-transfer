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
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"tkestack.io/image-transfer/pkg/log"
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

		manifestSchemaListObj, err := manifest.Schema2ListFromManifest(m)
		if err != nil {
			return nil, err
		}

		var subManifestInfoSlice []manifest.Manifest

		for _, manifestDescriptorElem := range manifestSchemaListObj.Manifests {

			log.Infof("handle manifest OS:%s Architecture:%s ", manifestDescriptorElem.Platform.OS,
				manifestDescriptorElem.Platform.Architecture)

			subManifestByte, subManifestType, err := i.source.GetManifest(i.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				log.Errorf("Get manifest %v of OS:%s Architecture:%s for manifest list error: %v",
					manifestDescriptorElem.Digest, manifestDescriptorElem.Platform.OS,
					manifestDescriptorElem.Platform.Architecture, err)
				return nil, err
			}

			subManifest, err := ManifestHandler(subManifestByte, subManifestType, i)
			if err != nil {
				log.Errorf("Handle sub manifest error: %v", err)
				return nil, err
			}

			if subManifest != nil {
				subManifestInfoSlice = append(subManifestInfoSlice, subManifest...)
			}
		}

		return subManifestInfoSlice, nil
	} else if t == specsv1.MediaTypeImageManifest {
		// Handle OCI Image Manifest
		manifestInfo, err := manifest.OCI1FromManifest(m)
		if err != nil {
			return nil, err
		}
		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, nil
	} else if t == specsv1.MediaTypeImageIndex {
		// Handle OCI Image Index
		ociIndexObj, err := manifest.OCI1IndexFromManifest(m)
		if err != nil {
			return nil, err
		}

		var subManifestInfoSlice []manifest.Manifest

		for _, descriptor := range ociIndexObj.Manifests {

			log.Infof("handle OCI manifest OS:%s Architecture:%s ", descriptor.Platform.OS,
				descriptor.Platform.Architecture)

			subManifestByte, subManifestType, err := i.source.GetManifest(i.ctx, &descriptor.Digest)
			if err != nil {
				log.Errorf("Get OCI manifest %v of OS:%s Architecture:%s for image index error: %v",
					descriptor.Digest, descriptor.Platform.OS,
					descriptor.Platform.Architecture, err)
				return nil, err
			}

			subManifest, err := ManifestHandler(subManifestByte, subManifestType, i)
			if err != nil {
				log.Errorf("Handle OCI sub manifest error: %v", err)
				return nil, err
			}

			if subManifest != nil {
				subManifestInfoSlice = append(subManifestInfoSlice, subManifest...)
			}
		}

		return subManifestInfoSlice, nil
	}
	return nil, fmt.Errorf("unsupported manifest type: %v", t)
}
