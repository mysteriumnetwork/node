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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

var pkgName = flag.String("pkg", "", "Same as abigen tool from ethereum project")
var output = flag.String("out", "", "Filename where to write generated code. Unspecified - stdout")
var localdir = flag.String("localdir", "", "Local dir in which to search for specified contracts (. means current) overrides github repo")
var contracts = flag.String("contracts", "", "Filename(s) separated by comma, of truffle compiled smart contract(s) (json format)")
var githubRepo = flag.String("githubrepo", "mysteriumnetwork/payments-smart-contracts", "github repository under which to search releases and attached smart contracts")

func main() {
	flag.Parse()
	if *pkgName == "" {
		fmt.Println("package name missing(--pkg)")
		os.Exit(-1)
	}

	smartContracts, err := parseTruffleArtifacts(*localdir, *githubRepo, *contracts)
	if err != nil {
		fmt.Println("Error parsing truffle output: ", err.Error())
		os.Exit(-1)
	}

	for _, smartContract := range smartContracts {
		genCode, err := bindSmartContract(smartContract, *pkgName)
		if err != nil {
			fmt.Println("Error binding smart contract: ", err.Error())
			os.Exit(-1)
		}
		if err := writeToOutput(smartContract.ContractName, genCode, *output); err != nil {
			fmt.Println("Error writing generated code: ", err.Error())
			os.Exit(-1)
		}
	}
}

func writeToOutput(fileName string, genCode, output string) error {
	var writer io.Writer
	if output != "" {
		if err := os.MkdirAll(output, 0755); err != nil { // >:)
			return err
		}
		file, err := os.Create(filepath.Join(output, fileName+".go"))
		defer file.Close()
		if err != nil {
			return err
		}
		if _, err := io.WriteString(file, licenseHeader); err != nil {
			return err
		}
		writer = file
	} else {
		writer = os.Stdout
		_, err := io.WriteString(writer, fmt.Sprintf("--- Smart contract: %s ---\n", fileName))
		if err != nil {
			return err
		}
	}

	if _, err := io.WriteString(writer, genCode); err != nil {
		return err
	}
	return nil
}

func bindSmartContract(smartContract truffleArtifact, pkgName string) (string, error) {
	genCode, err := bind.Bind([]string{smartContract.ContractName}, []string{smartContract.AbiString()}, []string{smartContract.Bytecode}, pkgName, bind.LangGo)
	if err != nil {
		return "", err
	}
	return genCode, nil
}

func parseTruffleArtifacts(localDir, githubRepo, contracts string) ([]truffleArtifact, error) {
	contractList := strings.Split(contracts, ",")

	if localDir != "" {
		return loadFromLocalDir(localDir, contractList)
	}
	return loadFromGitRepo(githubRepo, contractList)
}

func loadFromGitRepo(githubRepo string, contractList []string) ([]truffleArtifact, error) {
	return nil, nil
}

func loadFromLocalDir(localDir string, contracts []string) ([]truffleArtifact, error) {
	var artifacts []truffleArtifact
	for _, contract := range contracts {
		reader, err := os.Open(filepath.Join(localDir, contract))
		if err != nil {
			return nil, err
		}
		artifact, err := parseTruffleArtifact(reader)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func parseTruffleArtifact(inputReader io.Reader) (truffleArtifact, error) {
	var output truffleArtifact
	if err := json.NewDecoder(inputReader).Decode(&output); err != nil {
		return truffleArtifact{}, err
	}
	return output, nil
}

type truffleArtifact struct {
	Bytecode     string          `json:"bytecode"`
	AbiBytes     json.RawMessage `json:"abi"`
	ContractName string          `json:"contractName"`
}

func (to truffleArtifact) AbiString() string {
	return string(to.AbiBytes)
}

const licenseHeader = `/*
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

`
