/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package check

import (
	"errors"
	"fmt"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/go-ci/commands"
	"github.com/mysteriumnetwork/go-ci/util"
	"github.com/mysteriumnetwork/node/ci/packages"
)

// Check performs commons checks.
func Check() {
	mg.Deps(CheckGenerate)
	mg.Deps(CheckSwagger)
	mg.Deps(CheckGoImports, CheckGoLint, CheckGoVet, CheckCopyright)
}

// CheckCopyright checks for copyright headers in files.
func CheckCopyright() error {
	return commands.CopyrightD(".", "pb", "tequilapi/endpoints/assets")
}

// CheckGoLint reports linting errors in the solution.
func CheckGoLint() error {
	return commands.GoLintD(".", "docs")
}

// CheckGoVet checks that the source is compliant with go vet.
func CheckGoVet() error {
	return commands.GoVet("./...")
}

// CheckGoImports checks for issues with go imports.
func CheckGoImports() error {
	return commands.GoImportsD(".", "pb", "tequilapi/endpoints/assets")
}

// CheckSwagger checks whether swagger spec at "tequilapi/docs/swagger.json" is valid against swagger specification 2.0.
func CheckSwagger() error {
	if err := sh.RunV("swagger", "validate", "tequilapi/docs/swagger.json"); err != nil {
		return fmt.Errorf("could not validate swagger spec: %w", err)
	}
	return nil
}

var checkGenerateExcludes = []string{
	"tequilapi/endpoints/assets/docs.go",
}

// CheckGenerate checks whether dynamic project parts are updated properly.
func CheckGenerate() error {
	filesBefore, err := getUncommittedFiles()
	fmt.Println("Uncommitted files (before):")
	fmt.Println(filesBefore)
	fmt.Println()

	mg.Deps(packages.Generate)

	filesAfter, err := getUncommittedFiles()
	if err != nil {
		return fmt.Errorf("could retrieve changed files: %w", err)
	}
	fmt.Println("Uncommitted files (after):")
	fmt.Println(filesAfter)
	fmt.Println()

	if len(filesBefore) != len(filesAfter) {
		fmt.Println(`Files below needs review with "mage generate"`)
		return errors.New("not all dynamic files are up-to-date")
	}

	fmt.Println("Dynamic files are up-to-date")
	return nil
}

func getUncommittedFiles() ([]string, error) {
	filesAll, err := sh.Output("git", "diff", "HEAD", "--name-only")
	if err != nil {
		return nil, fmt.Errorf("could not retrieve changed files: %w", err)
	}

	files := make([]string, 0)
	for _, file := range strings.Split(filesAll, "\n") {
		if file == "" {
			continue
		}
		if util.IsPathExcluded(checkGenerateExcludes, file) {
			continue
		}
		files = append(files, file)
	}

	return files, nil
}
