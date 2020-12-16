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

package utils

import (
	"fmt"
	"strings"
)

// The RepoURL will divide a images url to <registry>/<namespace>/<repo>:<tag>
type RepoURL struct {
	// origin url
	url string

	registry  string
	namespace string
	repo      string
	tag       string
}

// NewRepoURL creates a RepoURL
func NewRepoURL(url string) (*RepoURL, error) {
	// split to registry/namespace/repoAndTag
	slice := strings.SplitN(url, "/", 3)

	var tag, repo string
	repoAndTag := slice[len(slice)-1]
	s := strings.Split(repoAndTag, ":")
	if len(s) > 2 {
		return nil, fmt.Errorf("invalid repository url: %v", url)
	} else if len(s) == 2 {
		repo = s[0]
		tag = s[1]
	} else {
		repo = s[0]
		tag = ""
	}

	if len(slice) == 3 {
		return &RepoURL{
			url:       url,
			registry:  slice[0],
			namespace: slice[1],
			repo:      repo,
			tag:       tag,
		}, nil
	} else if len(slice) == 2 {
		// if first string is a domain
		if strings.Contains(slice[0], ".") {
			return &RepoURL{
				url:       url,
				registry:  slice[0],
				namespace: "",
				repo:      repo,
				tag:       tag,
			}, nil
		}

		return &RepoURL{
			url:       url,
			registry:  "registry.hub.docker.com",
			namespace: slice[0],
			repo:      repo,
			tag:       tag,
		}, nil
	} else {
		return &RepoURL{
			url:       url,
			registry:  "registry.hub.docker.com",
			namespace: "library",
			repo:      repo,
			tag:       tag,
		}, nil
	}
}

// GetURL returns the whole url
func (r *RepoURL) GetURL() string {
	url := r.GetURLWithoutTag()
	if r.tag != "" {
		url = url + ":" + r.tag
	}
	return url
}

// GetOriginURL returns the whole url
func (r *RepoURL) GetOriginURL() string {
	return r.url
}

// GetRegistry returns the registry in a url
func (r *RepoURL) GetRegistry() string {
	return r.registry
}

// GetNamespace returns the namespace in a url
func (r *RepoURL) GetNamespace() string {
	return r.namespace
}

// GetRepo returns the repository in a url
func (r *RepoURL) GetRepo() string {
	return r.repo
}

// GetTag returns the tag in a url
func (r *RepoURL) GetTag() string {
	return r.tag
}

// GetRepoWithNamespace returns namespace/repository in a url
func (r *RepoURL) GetRepoWithNamespace() string {
	if r.namespace == "" {
		return r.repo
	}
	return r.namespace + "/" + r.repo
}

// GetRepoWithTag returns repository:tag in a url
func (r *RepoURL) GetRepoWithTag() string {
	if r.tag == "" {
		return r.repo
	}
	return r.repo + ":" + r.tag
}

// GetURLWithoutTag returns registry/namespace/repository in a url
func (r *RepoURL) GetURLWithoutTag() string {
	if r.namespace == "" {
		return r.registry + "/" + r.repo
	}
	return r.registry + "/" + r.namespace + "/" + r.repo
}

// CheckIfIncludeTag checks if a repository string includes tag
func CheckIfIncludeTag(repository string) bool {
	return strings.Contains(repository, ":")
}

// IsContain judge the item is in items or not
func IsContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}
