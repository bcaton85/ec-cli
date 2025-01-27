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

package utils

import (
	"bytes"
	"path/filepath"
	"unicode"

	"github.com/ghodss/yaml"
	"github.com/spf13/afero"
)

var (
	AppFS        = afero.NewOsFs()
	CreateTmpDir = afero.TempDir
)

// ToJSON converts a single YAML document into a JSON document
// or returns an error. If the document appears to be JSON the
// YAML decoding path is not used.
func ToJSON(data []byte) ([]byte, error) {
	if hasJSONPrefix(data) {
		return data, nil
	}
	return yaml.YAMLToJSON(data)
}

var jsonPrefix = []byte("{")

// hasJSONPrefix returns true if the provided buffer appears to start with
// a JSON open brace.
func hasJSONPrefix(buf []byte) bool {
	return hasPrefix(buf, jsonPrefix)
}

// hasPrefix returns true if the first non-whitespace bytes in buf is prefix.
func hasPrefix(buf []byte, prefix []byte) bool {
	trim := bytes.TrimLeftFunc(buf, unicode.IsSpace)
	return bytes.HasPrefix(trim, prefix)
}

// CreateWorkDir creates the working directory in tmp and some subdirectories
func CreateWorkDir() (string, error) {
	workDir, err := CreateTmpDir(AppFS, afero.GetTempDir(AppFS, ""), "ec-work-")
	if err != nil {
		return "", err
	}

	// Create top level directories for Conftest
	for _, d := range []string{
		"policy",
		"data",
		// Later maybe
		//"input",
	} {
		err := AppFS.Mkdir(filepath.Join(workDir, d), 0o755)
		if err != nil {
			return "", err
		}
	}

	return workDir, nil
}
