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

package imagetransfer


import (
	"fmt"
	"tkestack.io/image-transfer/pkg/log"
	"tkestack.io/image-transfer/pkg/image-transfer/options"
	"github.com/spf13/cobra"
	flagUtil "tkestack.io/image-transfer/pkg/flag"
	"os"
)


// RunFunc defines the alias of the command entry function
type RunFunc func(cmd *cobra.Command, args []string)

// NewImageTransferCommand creates a *cobra.Command object with default parameters
func NewImageTransferCommand(basename string) *cobra.Command {
	flagUtil.InitFlags()

	opts := options.NewClientOptions()
	cmd := &cobra.Command{
		Use:  basename,
		Long: "image-transfer",
		Run:  run(opts),
	}

	opts.AddFlags(cmd.Flags())
	log.AddFlags(cmd.Flags())
	return cmd
}

func run(opts *options.ClientOptions) RunFunc {
	return func(cmd *cobra.Command, args []string) {
		log.InitLogger()
		defer log.FlushLogger()

		flagUtil.PrintFlags(cmd.Flags())


		client, err := NewTransferClient(opts)
		if err != nil {
			log.Errorf("init Transfer Client error: %v", err)
			os.Exit(1)
		}

		if err := client.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

	}
}
