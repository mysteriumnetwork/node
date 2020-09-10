/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package packages

import (
	"fmt"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/go-ci/util"
)

// Generate recreates dynamic project parts which changes time to time.
func Generate() {
	mg.Deps(GenerateProtobuf, GenerateSwagger)
}

// GenerateProtobuf generates Protobuf models.
func GenerateProtobuf() error {
	if err := sh.Run("protoc", "-I=.", "--go_out=./pb", "./pb/ping.proto"); err != nil {
		return err
	}
	if err := sh.Run("protoc", "-I=.", "--go_out=./pb", "./pb/p2p.proto"); err != nil {
		return err
	}
	if err := sh.Run("protoc", "-I=.", "--go_out=./pb", "./pb/session.proto"); err != nil {
		return err
	}
	return sh.Run("protoc", "-I=.", "--go_out=./pb", "./pb/payment.proto")
}

// GenerateSwagger generates Tequilapi documentation.
func GenerateSwagger() error {
	mg.Deps(GetSwagger)

	return sh.RunV("swagger", "generate", "spec", "-o", "tequilapi.json", "--scan-models")
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
