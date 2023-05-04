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
	"net/http"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/shurcooL/vfsgen"
)

// Generate recreates dynamic project parts which changes time to time.
func Generate() {
	mg.Deps(GenerateProtobuf, GenerateSwagger)

	// Doc generation should occur after swagger generation
	mg.Deps(GenerateDocs)
}

// GenerateProtobuf generates Protobuf models.
func GenerateProtobuf() error {
	mg.Deps(GetProtobuf)

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

// GetProtobuf installs protobuf golang compiler.
func GetProtobuf() error {
	err := sh.RunV("go", "install", "google.golang.org/protobuf/cmd/protoc-gen-go@v1.25.0")
	if err != nil {
		fmt.Println("could not go get 'protoc-gen-go'")
		return err
	}
	return nil
}

// GenerateSwagger Tequilapi Swagger specification.
func GenerateSwagger() error {
	mg.Deps(GetSwagger)

	return sh.RunV("swagger", "generate", "spec", "-o", "tequilapi/docs/swagger.json", "--scan-models", "-x", `\Agithub\.com/mysteriumnetwork/feedback(/[^/]*)*\z`)
}

// GenerateDocs generates Tequilapi documentation pages.
// Based on Redoc template for swagger - https://github.com/Redocly/redoc.
func GenerateDocs() error {
	err := vfsgen.Generate(
		http.Dir("./tequilapi/docs"),
		vfsgen.Options{
			Filename:     "tequilapi/endpoints/assets/docs.go",
			PackageName:  "assets",
			VariableName: "DocsAssets",
		},
	)
	if err != nil {
		return fmt.Errorf("could not generate documentation assets: %w", err)
	}
	return nil
}

// GetSwagger installs swagger tool.
func GetSwagger() error {
	err := sh.RunV("go", "install", "github.com/go-swagger/go-swagger/cmd/swagger@v0.30.4")
	if err != nil {
		fmt.Println("could not go get swagger")
		return err
	}
	return nil
}
