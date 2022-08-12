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
	"net"
	"strings"
	"sync"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/mysteriumnetwork/go-ci/commands"
	"github.com/mysteriumnetwork/go-ci/util"
	"github.com/mysteriumnetwork/node/ci/packages"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/requests"
)

// Check performs commons checks.
func Check() {
	mg.Deps(CheckGenerate)
	mg.Deps(CheckSwagger)
	mg.Deps(CheckGoImports, CheckDNSMaps, CheckGoLint, CheckGoVet, CheckCopyright)
}

// CheckCopyright checks for copyright headers in files.
func CheckCopyright() error {
	return commands.CopyrightD(".", "pb", "tequilapi/endpoints/assets", "supervisor/daemon/wireguard/wginterface/firewall")
}

// CheckGoLint reports linting errors in the solution.
func CheckGoLint() error {
	return commands.GoLintD(".", "docs", "services/wireguard/endpoint/netstack")
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

// CheckDNSMaps checks if the given dns maps in the metadata configuration actually point to the given IPS.
func CheckDNSMaps() error {
	ipMissmatches := make([]string, 0)

	valuesToCheck := make(map[string][]string, 0)
	for k, v := range metadata.MainnetDefinition.DNSMap {
		valuesToCheck[k] = v
	}

	for k, v := range valuesToCheck {
		ips, err := net.LookupIP(k)
		if err != nil {
			return err
		}

		found := false

		for _, ip := range v {
			for i := range ips {
				if ips[i].String() == ip {
					found = true
					break
				}
			}

			if found {
				break
			}
		}

		if !found {
			ipMissmatches = append(ipMissmatches, fmt.Sprintf("ip: %v, host: %v", ips, k))
		}

	}

	if len(ipMissmatches) > 0 {
		for _, v := range ipMissmatches {
			fmt.Println(v)
		}
		return errors.New("IP missmatches found in DNS hosts")
	}

	return nil
}

var checkGenerateExcludes = []string{
	"tequilapi/endpoints/assets/docs.go",
}

// CheckGenerate checks whether dynamic project parts are updated properly.
func CheckGenerate() error {
	filesBefore, err := getUncommittedFiles()
	if err != nil {
		return fmt.Errorf("could retrieve uncommitted files: %w", err)
	}
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

// CheckIPFallbacks checks if all the IP fallbacks are working correctly.
func CheckIPFallbacks() error {
	c := requests.NewHTTPClient("0.0.0.0", time.Second*30)
	wg := sync.WaitGroup{}
	wg.Add(len(ip.IPFallbackAddresses))

	res := make([]fallbackResult, len(ip.IPFallbackAddresses))
	for i, v := range ip.IPFallbackAddresses {
		go func(idx int, url string) {
			defer wg.Done()

			r, err := ip.RequestAndParsePlainIPResponse(c, url)
			res[idx] = fallbackResult{
				url:      url,
				response: r,
				err:      err,
			}
		}(i, v)
	}
	wg.Wait()

	// check that no errors are present
	var errorsPresent bool
	for i := range res {
		if res[i].err != nil {
			fmt.Printf("Unexpected error for %v: %v\n", res[i].url, res[i].err.Error())
			errorsPresent = true
		}
	}

	if errorsPresent {
		return errors.New("unexpected errors present")
	}

	initialResult := res[0].response
	for i := range res {
		if res[i].response != initialResult {
			fmt.Println("Missmatch for ips!")

			// print all results
			for j := range res {
				fmt.Printf("%v: %v\n", res[j].url, res[j].response)
			}

			return errors.New("ip missmatches found")
		}
	}

	fmt.Println("All IPS are in order")
	return nil
}

type fallbackResult struct {
	url      string
	response string
	err      error
}
