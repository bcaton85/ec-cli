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

package replacer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/hacbs-contract/ec-cli/internal/image"
)

func Replace(ctx context.Context, images []string, source string, overwrite bool, opts *CatalogOptions) ([]byte, error) {
	resolvedImages, err := image.ParseAndResolveAll(images)
	if err != nil {
		return nil, err
	}

	replacers := make([]imageReplacer, 0, len(resolvedImages)+1)
	for _, image := range resolvedImages {
		r, err := newBasicImageReplacer(image)
		if err != nil {
			return nil, err
		}
		replacers = append(replacers, r)
	}
	catalogReplacer, err := newCatalogImageReplacer(opts)
	if err != nil {
		return nil, err
	}
	replacers = append(replacers, catalogReplacer)

	schema, value, found := strings.Cut(source, "://")
	if !found {
		schema, value = "file", schema
	}
	switch schema {
	case "file":
		return replaceFile(value, replacers, overwrite)
	case "git":
		return replaceGitFiles(ctx, value, replacers)
	case "https", "http":
		// Use original source string
		return replaceGitFiles(ctx, source, replacers)
	default:
		return nil, fmt.Errorf("%s is not a valid source schema", schema)
	}
}

type CatalogOptions struct {
	CatalogName string
	RepoBase    string
	HubAPIURL   string
}

func replaceFile(filename string, replacers []imageReplacer, overwrite bool) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	writer := bytes.NewBuffer(nil)
	for scanner.Scan() {
		line := scanner.Bytes()
		for _, replace := range replacers {
			if replace.match(line) {
				line = replace.replace(line)
			}
		}
		if _, err := writer.Write(line); err != nil {
			return nil, err
		}
		if _, err = writer.WriteString("\n"); err != nil {
			return nil, err
		}
	}

	out := writer.Bytes()

	if overwrite {
		stat, err := os.Stat(filename)
		if err != nil {
			return nil, err
		}
		if err := ioutil.WriteFile(filename, out, stat.Mode()); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// cloneRepo is used as an alias for git.PlainCloneContext in order to facilitate testing
var cloneRepo = git.PlainCloneContext

func replaceGitFiles(ctx context.Context, gitRef string, replacers []imageReplacer) ([]byte, error) {
	dir, err := os.MkdirTemp("", "ec-replace-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	repoName, branch, found := strings.Cut(gitRef, "#")
	if !found || branch == "" {
		branch = "main"
	}

	cloneOptions := &git.CloneOptions{
		URL:           repoName,
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	}
	if _, err = cloneRepo(ctx, dir, false, cloneOptions); err != nil {
		return nil, err
	}

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, readErr error) error {
		if readErr != nil {
			return readErr
		}
		if d.IsDir() {
			return nil
		}
		if ext := strings.ToLower(filepath.Ext(path)); ext != ".yaml" && ext != ".yml" {
			return nil
		}

		if _, err := replaceFile(path, replacers, true); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	changes, err := getGitChanges(dir)
	if err != nil {
		return nil, err
	}
	return changes, nil
}

// getGitChanges returns the changes associated with the commit hash
// for the git repository.
func getGitChanges(dir string) ([]byte, error) {
	// This is not ideal, but there are two issues with using go-git
	// for this.
	// First, it is currently not possible to produce a diff for
	// uncommitted changes. See related discussion:
	//		https://github.com/src-d/go-git/issues/604
	// Second, the output generated by go-git is just plain wrong and
	// in some cases does not work with "git apply".
	cmd := exec.Command("git", "-C", dir, "diff")
	var changes bytes.Buffer
	cmd.Stdout = &changes
	if err := cmd.Run(); err != nil {
		return []byte{}, err
	}
	return changes.Bytes(), nil
}
