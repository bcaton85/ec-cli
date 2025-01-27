// Copyright 2022 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

// Define the `ec fetch policy` command
package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/hacbs-contract/ec-cli/internal/policy/source"
	"github.com/hacbs-contract/ec-cli/internal/utils"
)

func fetchPolicyCmd() *cobra.Command {
	var (
		sourceUrls []string
		destDir    string
		useWorkDir bool
	)

	cmd := &cobra.Command{
		Use:   "policy --source <source-url>",
		Short: "Fetch policy rules from a git repository or other source",
		Long: `Fetch policy rules (rego files) from a git repository or other source.

Each policy source will be downloaded into a separate unique directory inside
the "policy" directory under the destination directory specified. The
destination directory is either an automatically generated temporary work dir
if --work-dir is set, the directory specified with the --dest flag, or the
current directory if neither flag is specified.

This command is based on 'conftest pull' so you can refer to the conftest pull
documentation for more usage examples and for details on the different types of
supported source URLs.

Note that this command is not typically required to verify the Enterprise
Contract. It has been made available for troubleshooting and debugging
purposes.`,

		Example: `Fetching policies from multiple sources to a specific directory:

  ec fetch policy --dest fetched-policies \
    --source github.com/hacbs-contract/ec-policies//policy/lib \
    --source github.com/hacbs-contract/ec-policies//policy/release

Fetching policies from multiple sources to an automatically generated temporary
work directory:

  ec fetch policy --work-dir \
    --source github.com/hacbs-contract/ec-policies//policy/lib \
    --source github.com/hacbs-contract/ec-policies//policy/release

Different style url formats are supported. In this example "policy" is treated as
a subdirectory even without the go-getter style // delimiter:

  ec fetch policy --source https://github.com/hacbs-contract/ec-policies/policy

Notes:

- The --dest flag will be ignored if --work-dir is set
- Adding a protocol prefix such as 'git::' to the source url forces it to be treated
  as a go-getter style url.`,

		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if useWorkDir {
				workDir, err := utils.CreateWorkDir()
				if err != nil {
					log.Debug("Failed to create work dir!")
					return err
				}
				destDir = workDir
			}

			for _, s := range sourceUrls {
				// Do everything the same way that it would be done when an image validation happens
				policyUrl := source.PolicyUrl(s)
				policySource := &policyUrl
				err := policySource.GetPolicies(cmd.Context(), destDir, true)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&sourceUrls, "source", "s", []string{}, "source url. multiple values are allowed")
	cmd.Flags().StringVarP(&destDir, "dest", "d", ".", "use the specified download destination directory. ignored if --work-dir is set")
	cmd.Flags().BoolVarP(&useWorkDir, "work-dir", "w", false, "use a temporary work dir as the download destination directory")

	if err := cmd.MarkFlagRequired("source"); err != nil {
		panic(err)
	}

	return cmd
}
