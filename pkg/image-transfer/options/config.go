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

package options

import (
	"github.com/spf13/pflag"
)

// ConfigOptions 基础配置信息
type ConfigOptions struct {
	SecurityFile string
	RuleFile string
	RoutineNums int
	RetryNums int
	QPS int
	DefaultRegistry string
	DefaultNamespace string
	CCRToTCR bool
	CCRRegion string
	TCRRegion string
	TCRName string
	SecretFile string
	// if target tag is exist override it
	TagExistOverridden bool
}

// NewConfigOptions creates a NewConfigOptions object with default
// parameters.
func NewConfigOptions() *ConfigOptions {
	return &ConfigOptions{}
}

// Validate is used to parse and validate the parameters entered by the user at
// the command line when the program starts.
func (o *ConfigOptions) Validate() []error {
	var allErrors []error

	return allErrors
}

// AddFlags adds flags related to authenticate for a specific APIServer to the
// specified FlagSet
func (o *ConfigOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.SecurityFile, "securityFile", o.SecurityFile,
		"Get registry auth config from config file path")
	fs.StringVar(&o.RuleFile, "ruleFile", o.RuleFile,
		"Get images rules config from config file path")
	fs.StringVar(&o.DefaultRegistry, "registry", o.DefaultRegistry,
		"default destinate registry url when destinate registry is not " +
		"given in the config file, can also be set with DEFAULT_REGISTRY environment value")
	fs.StringVar(&o.DefaultNamespace, "ns", o.DefaultNamespace,
		"default destinate namespace when destinate namespace is not" +
		" given in the config file, can also be set with DEFAULT_NAMESPACE environment value")
	fs.IntVar(&o.RoutineNums, "routines", 5,
		"number of goroutines, default value is 5, max routines is 200")
	fs.IntVar(&o.RetryNums, "retry", 2,
		"number of retries, default value is 2")
	fs.IntVar(&o.QPS, "qps", 100,
		"QPS of request, default value is 100, max is 30000")
	fs.BoolVar(&o.CCRToTCR, "ccrToTcr", false,
		"mode: transfer ccr images to tcr, default value is false")
	fs.StringVar(&o.CCRRegion, "ccrRegion", "ap-guangzhou",
		"ccr region, default value is ap-guangzhou. this flag is used when flag ccrToTcr=true")
	fs.StringVar(&o.TCRRegion, "tcrRegion", "ap-guangzhou",
		"tcr region, default value is ap-guangzhou. this flag is used when flag ccrToTcr=true")
	fs.StringVar(&o.TCRName, "tcrName", o.TCRName,
		"tcr name. this flag is used when flag ccrToTcr=true")
	fs.StringVar(&o.SecretFile, "secretFile", o.SecretFile,
		"Tencent Cloud secretId 、secretKey for access ccr and tcr. this flag is used when flag ccrToTcr=true")
	fs.BoolVar(&o.TagExistOverridden, "tag-exist-overridden", true, "if target tag is exist, override it")
}
