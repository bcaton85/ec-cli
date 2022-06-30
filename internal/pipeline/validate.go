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
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.`
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"context"

	"github.com/hacbs-contract/ec-cli/internal/evaluation_target/pipeline_definition_file"
	"github.com/hacbs-contract/ec-cli/internal/output"
	"github.com/hacbs-contract/ec-cli/internal/policy_source"
)

//ValidatePipeline calls NewPipelineEvaluator to obtain an PipelineEvaluator. It then executes the associated TestRunner
//which tests the associated pipeline file(s) against the associated policies, and displays the output.
func ValidatePipeline(ctx context.Context, fpath string, policyRepo policy_source.PolicyRepo, namespace string) (*output.Output, error) {
	p, err := pipeline_definition_file.NewPipelineDefinitionFile(ctx, fpath, policyRepo, namespace)
	if err != nil {
		return nil, err
	}

	results, err := p.Evaluator.TestRunner.Run(p.Evaluator.Context, []string{p.Fpath})
	if err != nil {
		return nil, err
	}
	return &output.Output{PolicyCheck: results}, nil
}
