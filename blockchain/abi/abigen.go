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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

var pkgName = flag.String("pkg", "", "Same as abigen tool from ethereum project")
var output = flag.String("out", "", "Filename where to write generated code. Unspecified - stdout")
var input = flag.String("in", "", "Filename of truffle compiled smart contract (json format)")

func main() {
	flag.Parse()
	if *pkgName == "" {
		fmt.Println("package name missing(--pkg)")
		os.Exit(-1)
	}

	if *input == "" {
		fmt.Println("input filename is missing")
		os.Exit(-1)
	}

	smartContract, err := parseTruffleArtifact(*input)
	if err != nil {
		fmt.Println("Error parsing truffle output: ", err.Error())
		os.Exit(-1)
	}

	genCode, err := bind.Bind([]string{smartContract.ContractName}, []string{smartContract.AbiString()}, []string{smartContract.Bytecode}, *pkgName, bind.LangGo)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(-1)
	}
	writer := os.Stdout
	if *output != "" {
		writer, err = os.Create(*output)
		if err != nil {
			fmt.Println("Error:", err.Error())
			os.Exit(1)
		}
		defer writer.Close()
	}
	_, err = io.WriteString(writer, genCode)
	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}

func parseTruffleArtifact(input string) (truffleArtifact, error) {
	reader, err := os.Open(input)
	if err != nil {
		return truffleArtifact{}, err
	}
	var output truffleArtifact
	err = json.NewDecoder(reader).Decode(&output)
	if err != nil {
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
