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
	"fmt"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/go-ci/commands"
	"github.com/mysteriumnetwork/go-ci/util"
)

// Check performs commons checks.
func Check() {
	mg.Deps(CheckSwagger)
	mg.Deps(CheckGoImports, CheckGoLint, CheckGoVet, CheckCopyright)
}

// CheckCopyright checks for copyright headers in files.
func CheckCopyright() error {
	return commands.Copyright("./...", "docs")
}

// CheckGoLint reports linting errors in the solution.
func CheckGoLint() error {
	return commands.GoLint("./...", "docs")
}

// CheckGoVet Checks that the source is compliant with go vet.
func CheckGoVet() error {
	return commands.GoVet("./...")
}

// CheckGoImports checks for issues with go imports.
func CheckGoImports() error {
	return commands.GoImports("./...", "docs")
}

// CheckSwagger checks whether swagger spec at "tequilapi.json" is valid against swagger specification 2.0.
func CheckSwagger() error {
	mg.Deps(GetSwagger)

	err := sh.RunV("swagger", "generate", "spec", "-o", "tequilapi.json", "--scan-models")
	if err != nil {
		fmt.Println("could not generate swagger spec")
		return err
	}

	if err := sh.RunV("swagger", "validate", "tequilapi.json"); err != nil {
		fmt.Println("could not validate swagger spec")
		return err
	}
	return nil
}

// GetSwagger installs swagger tool.
func GetSwagger() error {
	path, _ := util.GetGoBinaryPath("swagger")
	if path != "" {
		fmt.Println("Tool 'swagger' already installed")
		return nil
	}
	err := sh.RunV("go", "get", "-u", "github.com/go-swagger/go-swagger/cmd/swagger")
	if err != nil {
		fmt.Println("could not go get swagger")
		return err
	}
	return nil
}
