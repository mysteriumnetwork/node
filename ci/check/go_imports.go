/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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
	"go/build"
	"os"
	"path"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/go-ci/shell"
	"github.com/mysteriumnetwork/go-ci/util"
)

// GoImportsD checks for issues with go imports.
//
// Instead of packages, it operates on directories, thus it is compatible with gomodules outside GOPATH.
//
// Example:
//
//	commands.GoImportsD(".", "docs")
func GoImportsD(dir string, excludes ...string) error {
	mg.Deps(GetImports)
	goimportsBin, err := util.GetGoBinaryPath("goimports")
	if err != nil {
		fmt.Println("❌ GoImports")
		fmt.Println("Tool 'goimports' not found")
		return err
	}
	var allExcludes []string
	allExcludes = append(allExcludes, excludes...)
	allExcludes = append(allExcludes, util.GoLintExcludes()...)
	dirs, err := util.GetProjectFileDirectories(allExcludes)
	if err != nil {
		return err
	}
	out, err := shell.NewCmd(goimportsBin + " -e -l -d " + strings.Join(dirs, " ")).Output()
	if err != nil {
		fmt.Println("❌ GoImports")
		fmt.Println("goimports: error executing")
		return err
	}
	if len(out) != 0 {
		fmt.Println("❌ GoImports")
		fmt.Println("goimports: the following files contain go import errors:")
		fmt.Println(out)
		return errors.New("goimports: not all imports follow the goimports format")
	}
	fmt.Println("✅ GoImports")
	return nil
}

// GetGoPath returns the go path
func GetGoPath() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	return gopath
}

// GetGoBinaryPath looks for the given binary in path, if not checks if it's in $GOPATH/bin
func GetGoBinaryPath(binaryName string) (string, error) {
	res, err := sh.Output("which", binaryName)
	if err == nil {
		return res, nil
	}
	gopath := GetGoPath()
	binaryUnderGopath := path.Join(gopath, "bin", binaryName)
	if _, err := os.Stat(binaryUnderGopath); os.IsNotExist(err) {
		return "", err
	}
	return binaryUnderGopath, nil
}

// GetImports installs goimports binary.
func GetImports() error {
	path, _ := GetGoBinaryPath("goimports")
	if path != "" {
		fmt.Println("Tool 'goimports' already installed")
		return nil
	}
	err := sh.RunV("go", "install", "golang.org/x/tools/cmd/goimports@v0.41.0")
	if err != nil {
		fmt.Println("Could not go get goimports")
		return err
	}
	return nil
}
