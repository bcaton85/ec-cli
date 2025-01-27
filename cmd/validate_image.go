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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"

	ecc "github.com/hacbs-contract/enterprise-contract-controller/api/v1alpha1"
	"github.com/hashicorp/go-multierror"
	appstudioshared "github.com/redhat-appstudio/managed-gitops/appstudio-shared/apis/appstudio.redhat.com/v1alpha1"
	"github.com/spf13/cobra"

	"github.com/hacbs-contract/ec-cli/internal/applicationsnapshot"
	"github.com/hacbs-contract/ec-cli/internal/output"
	"github.com/hacbs-contract/ec-cli/internal/policy"
)

type imageValidationFunc func(ctx context.Context, imageRef string, policy *ecc.EnterpriseContractPolicySpec) (*output.Output, error)

func validateImageCmd(validate imageValidationFunc) *cobra.Command {
	var data = struct {
		policyConfiguration string
		imageRef            string
		publicKey           string
		rekorURL            string
		strict              bool
		input               string
		filePath            string
		output              string
		spec                *appstudioshared.ApplicationSnapshotSpec
		policy              *ecc.EnterpriseContractPolicySpec
	}{
		policyConfiguration: "ec-policy",
	}
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Validate conformance of container images with the Enterprise Contract",
		Long: `Validate conformance of container images with the Enterprise Contract

For each image, validation is performed in stages to determine if the image
conforms to the Enterprise Contract.

The first validation stage determines if an image has been signed, and the
signature matches the provided public key. This is akin to the "cosign verify"
command.

The second validation stage determines if one or more attestations exist, and
those attestations have been signed matching the provided public key, similarly
to the "cosign verify-attestation" command. This stage temporarily stores the
attestations for usage in the next stage.

The final stage verifies the attestations conform to rego policies defined in
the EnterpriseContractPolicy.

Validation advances each stage as much as possible for each image in order to
capture all issues in a single execution.`,
		Example: `Validate single image with the default policy defined in the
EnterpriseContractPolicy custom resource named "ec-policy" in the current
Kubernetes namespace:

  ec validate image --image registry/name:tag

Validate multiple images from an ApplicationSnapshot Spec file:

  ec validate image --file-path my-app.yaml

Validate attestation of images from an inline ApplicationSnapshot Spec:

  ec validate image --json-input '{"components":[{"containerImage":"<image url>"}]}'

Use a different public key than the one from the EnterpriseContractPolicy resource:

  ec validate image --image registry/name:tag --public-key <path/to/public/key>

Use a different Rekor URL than the one from the EnterpriseContractPolicy resource:

  ec validate image --image registry/name:tag --rekor-url https://rekor.example.org

Return a non-zero status code on validation failure:

  ec validate image --image registry/name:tag --strict

Use an EnterpriseContractPolicy resource from the currently active kubernetes context:

  ec validate image --image registry/name:tag --policy my-policy

Use an EnterpriseContractPolicy resource from a different namespace:

  ec validate image --image registry/name:tag --policy my-namespace/my-policy

Use an inline EnterpriseContractPolicy spec
  ec validate image --image registry/name:tag --policy '{"publicKey": "<path/to/public/key>"}'`,
		PreRunE: func(cmd *cobra.Command, args []string) (allErrors error) {
			if s, err := applicationsnapshot.DetermineInputSpec(
				data.filePath, data.input, data.imageRef,
			); err != nil {
				allErrors = multierror.Append(allErrors, err)
			} else {
				data.spec = s
			}

			if policy, err := policy.NewPolicy(
				cmd.Context(), data.policyConfiguration, data.rekorURL, data.publicKey,
			); err != nil {
				allErrors = multierror.Append(allErrors, err)
			} else {
				data.policy = policy
			}

			return
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			type result struct {
				err       error
				component applicationsnapshot.Component
			}

			appComponents := data.spec.Components

			ch := make(chan result, len(appComponents))

			var lock sync.WaitGroup
			for _, c := range appComponents {
				lock.Add(1)
				go func(comp appstudioshared.ApplicationSnapshotComponent) {
					defer lock.Done()

					out, err := validate(cmd.Context(), comp.ContainerImage, data.policy)
					res := result{
						err: err,
						component: applicationsnapshot.Component{
							ApplicationSnapshotComponent: appstudioshared.ApplicationSnapshotComponent{
								Name:           comp.Name,
								ContainerImage: comp.ContainerImage,
							},
							Success: err == nil,
						},
					}

					// Skip on err to not panic. Error is return on routine completion.
					if err == nil {
						res.component.Violations = out.Violations()
						res.component.Warnings = out.Warnings()
					}
					res.component.Success = err == nil && len(res.component.Violations) == 0

					ch <- res
				}(c)
			}

			lock.Wait()
			close(ch)

			var components []applicationsnapshot.Component
			var allErrors error = nil
			for r := range ch {
				if r.err != nil {
					e := fmt.Errorf("error validating image %s of component %s: %w", r.component.ContainerImage, r.component.Name, r.err)
					allErrors = multierror.Append(allErrors, e)
				} else {
					components = append(components, r.component)
				}
			}

			report, err, success := applicationsnapshot.Report(components)
			if allErrors != nil {
				return multierror.Append(allErrors, err)
			}

			if len(data.output) > 0 {
				if err := ioutil.WriteFile(data.output, []byte(report), 0644); err != nil {
					return multierror.Append(allErrors, err)
				}
				fmt.Printf("Report written to %s\n", data.output)
			} else {
				_, err := cmd.OutOrStdout().Write([]byte(report))
				if err != nil {
					return multierror.Append(allErrors, err)
				}
			}

			if data.strict && !success {
				// TODO: replace this with proper message and exit code 1.
				return errors.New("success criteria not met")
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&data.policyConfiguration, "policy", "p", data.policyConfiguration,
		"EntepriseContractPolicy reference [<namespace>/]<name>")
	cmd.Flags().StringVarP(&data.imageRef, "image", "i", data.imageRef, "OCI image reference")
	cmd.Flags().StringVarP(&data.publicKey, "public-key", "k", data.publicKey,
		"path to the public key. Overrides publicKey from EnterpriseContractPolicy")
	cmd.Flags().StringVarP(&data.rekorURL, "rekor-url", "r", data.rekorURL,
		"Rekor URL. Overrides rekorURL from EnterpriseContractPolicy")
	cmd.Flags().StringVarP(&data.filePath, "file-path", "f", data.filePath,
		"path to ApplicationSnapshot Spec JSON file")
	cmd.Flags().StringVarP(&data.input, "json-input", "j", data.input,
		"JSON represenation of an ApplicationSnapshot Spec")
	cmd.Flags().StringVarP(&data.output, "output-file", "o", data.output,
		"write output to a file. Use empty string for stdout, default behavior")
	cmd.Flags().BoolVarP(&data.strict, "strict", "s", data.strict,
		"return non-zero status on non-successful validation")

	if len(data.input) > 0 || len(data.filePath) > 0 {
		if err := cmd.MarkFlagRequired("image"); err != nil {
			panic(err)
		}
	}
	return cmd
}
