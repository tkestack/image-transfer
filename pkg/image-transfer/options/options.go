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

// ClientOptions contains the configuration required for api server to run.
type ClientOptions struct {
	Config *ConfigOptions
}

// NewClientOptions creates a new ClientOptions object with default
// parameters.
func NewClientOptions() *ClientOptions {
	return &ClientOptions{
		Config: NewConfigOptions(),
	}
}

// AddFlags adds flags for a specific api server to the specified FlagSet object.
func (o *ClientOptions) AddFlags(fs *pflag.FlagSet) {
	o.Config.AddFlags(fs)
}

// Validate checks APIServerOptions and return a slice of found errors.
func (o *ClientOptions) Validate() []error {
	var errors []error



	if errs := o.Config.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}


	return errors
}
