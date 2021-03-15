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
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/pkg/blobinfocache/memory"
	"github.com/containers/image/v5/pkg/blobinfocache/none"
	"github.com/pkg/errors"
	"tkestack.io/image-transfer/pkg/log"
)

var (
	// NoCache used to disable a blobinfocache
	NoCache = none.NoCache
	// Memory cache to enable a blobinfocache
	Memory = memory.New()
)

// Job act as a sync action, it will pull a images from source to target
type Job struct {
	Source *ImageSource
	Target *ImageTarget
}

// NewJob creates a transfer job
func NewJob(source *ImageSource, target *ImageTarget) *Job {

	return &Job{
		Source: source,
		Target: target,
	}
}

// Run is the main function of a transfer job
func (j *Job) Run() error {
	// get manifest from source
	manifestByte, manifestType, err := j.Source.GetManifest()
	if err != nil {
		log.Errorf("Failed to get manifest from %s/%s:%s error: %v",
			j.Source.GetRegistry(), j.Source.GetRepository(), j.Source.GetTag(), err)
		return err
	}
	log.Infof("Get manifest from %s/%s:%s", j.Source.GetRegistry(), j.Source.GetRepository(), j.Source.GetTag())

	blobInfos, err := j.Source.GetBlobInfos(manifestByte, manifestType)
	if err != nil {
		log.Errorf("Get blob info from %s/%s:%s error: %v",
			j.Source.GetRegistry(), j.Source.GetRepository(), j.Source.GetTag(), err)
		return err
	}

	// blob transformation
	for _, blobinfo := range blobInfos {
		blobExist, err := j.Target.CheckBlobExist(blobinfo)
		if err != nil {
			log.Errorf("Check blob %s(%v) to %s/%s:%s exist error: %v",
				blobinfo.Digest, blobinfo.Size, j.Target.GetRegistry(), j.Target.GetRepository(), j.Target.GetTag(), err)
			return err
		}

		if !blobExist {
			// pull a blob from source
			log.Infof("Getting blob from %s/%s:%s ing...", j.Source.GetRegistry(), j.Source.GetRepository(), j.Source.GetTag())
			blob, size, err := j.Source.GetABlob(blobinfo)
			if err != nil {
				log.Errorf("Get blob %s(%v) from %s/%s:%s failed: %v", blobinfo.Digest,
					size, j.Source.GetRegistry(), j.Source.GetRepository(), j.Source.GetTag(), err)
				return err
			}

			log.Infof("Get a blob %s(%v) from %s/%s:%s success", blobinfo.Digest, size,
				j.Source.GetRegistry(), j.Source.GetRepository(), j.Source.GetTag())

			blobinfo.Size = size
			// push a blob to target
			log.Infof("Putting blob to %s/%s:%s ing...", j.Target.GetRegistry(), j.Target.GetRepository(), j.Target.GetTag())
			if err := j.Target.PutABlob(blob, blobinfo); err != nil {
				log.Errorf("Put blob %s(%v) to %s/%s:%s failed: %v", blobinfo.Digest, blobinfo.Size,
					j.Target.GetRegistry(), j.Target.GetRepository(), j.Target.GetTag(), err)
				if closeErr := blob.Close(); closeErr != nil {
					return errors.Wrapf(err, " (close error: %v)", closeErr)
				}
				return err
			}

			log.Infof("Put blob %s(%v) to %s/%s:%s success", blobinfo.Digest, blobinfo.Size,
				j.Target.GetRegistry(), j.Target.GetRepository(), j.Target.GetTag())
		} else {
			// print the log of ignored blob
			log.Infof("Blob %s(%v) has been pushed to %s, will not be pulled", blobinfo.Digest,
				blobinfo.Size, j.Target.GetRegistry()+"/"+j.Target.GetRepository())
		}
	}

	//Push manifest list
	if manifestType == manifest.DockerV2ListMediaType {
		manifestSchemaListInfo, err := manifest.Schema2ListFromManifest(manifestByte)
		if err != nil {
			return err
		}

		var subManifestByte []byte

		// push manifest to target
		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {

			log.Infof("handle manifest OS:%s Architecture:%s ", manifestDescriptorElem.Platform.OS,
				manifestDescriptorElem.Platform.Architecture)

			subManifestByte, _, err = j.Source.source.GetManifest(j.Source.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				log.Errorf("Get manifest %v of OS:%s Architecture:%s for manifest list error: %v",
					manifestDescriptorElem.Digest, manifestDescriptorElem.Platform.OS,
					manifestDescriptorElem.Platform.Architecture, err)
				return err
			}

			if err := j.Target.PushManifest(subManifestByte); err != nil {
				log.Errorf("Put manifest to %s/%s:%s error: %v", j.Target.GetRegistry(),
					j.Target.GetRepository(), j.Target.GetTag(), err)
				return err
			}

			log.Infof("Put manifest to %s/%s:%s os:%s arch:%s", j.Target.GetRegistry(), j.Target.GetRepository(),
				j.Target.GetTag(), manifestDescriptorElem.Platform.OS, manifestDescriptorElem.Platform.Architecture)

		}

		// push manifest list to target
		if err := j.Target.PushManifest(manifestByte); err != nil {
			log.Errorf("Put manifestList to %s/%s:%s error: %v", j.Target.GetRegistry(),
				j.Target.GetRepository(), j.Target.GetTag(), err)
			return err
		}

		log.Infof("Put manifestList to %s/%s:%s", j.Target.GetRegistry(), j.Target.GetRepository(), j.Target.GetTag())

	} else {

		// push manifest to target
		if err := j.Target.PushManifest(manifestByte); err != nil {
			log.Errorf("Put manifest to %s/%s:%s error: %v", j.Target.GetRegistry(),
				j.Target.GetRepository(), j.Target.GetTag(), err)
			return err
		}

		log.Infof("Put manifest to %s/%s:%s", j.Target.GetRegistry(), j.Target.GetRepository(), j.Target.GetTag())
	}

	log.Infof("Synchronization successfully from %s/%s:%s to %s/%s:%s", j.Source.GetRegistry(), j.Source.GetRepository(),
		j.Source.GetTag(), j.Target.GetRegistry(), j.Target.GetRepository(), j.Target.GetTag())

	return nil
}
