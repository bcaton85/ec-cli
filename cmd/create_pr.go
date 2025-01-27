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

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/hacbs-contract/ec-cli/internal/ecgit"
)

func createPR() *cobra.Command {
	var data = struct {
		componentRepoURL  string
		destinationBranch string
		prBranchName      string
		prTitle           string
		prBody            string
		patchFilePath     string
	}{
		destinationBranch: "main",
		prTitle:           "Enterprise Contract Automated Update",
		prBody:            "This is an automated PR generated by Enterprise Contract",
	}
	cmd := &cobra.Command{
		Use:   "create-pr",
		Short: "Create a GitHub pull request based on source and target branches",
		Long: `Create a GitHub pull request based on source and target branches

The following steps are performed to create a pull request:
  - Clone the provided remote repository (--repo)
  - Create a branch with specified name (--branch-name)
  - Apply the given patch file (--patch)
  - Add and commit any files changes
  - Push newly created branch to the remote repository
  - Create a pull request for the target branch (--desitination-branch)

The environment variables GITHUB_USERNAME and GITHUB_TOKEN must be set accordingly.`,
		Example: `  ec-cli create-pr --repo https://github.com/example/repo --branch-name <new branch name> --patch <path/to/patch/file>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ecgit.CreateAutomatedPR(cmd.Context(), data.componentRepoURL, data.patchFilePath, data.destinationBranch, data.prBranchName, data.prTitle, data.prBody)
		},
	}
	cmd.Flags().StringVar(&data.componentRepoURL, "repo", data.componentRepoURL, "repo URL")
	cmd.Flags().StringVar(&data.destinationBranch, "destination-branch", data.destinationBranch, "target branch for PR")
	cmd.Flags().StringVar(&data.prBranchName, "branch-name", data.prBranchName, "branch to create with patch changes")
	cmd.Flags().StringVar(&data.prTitle, "title", data.prTitle, "title of the PR")
	cmd.Flags().StringVar(&data.prBody, "body", data.prBody, "body of the PR")
	cmd.Flags().StringVar(&data.patchFilePath, "patch", data.patchFilePath, "path to the patch file")
	return cmd
}

func init() {
	create := createPR()
	RootCmd.AddCommand(create)
}
