// +build integration

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

package testinstall

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/magefile/mage/sh"
	"github.com/stretchr/testify/assert"
)

const (
	installURL  = "https://raw.githubusercontent.com/mysteriumnetwork/node/master/install.sh"
	installFile = "install.sh"
)

// TestInstall tests installation on various Linux distributions.
// SystemD docker images based on https://github.com/j8r/dockerfiles
// Installs myst using install.sh, tests for healthcheck output.
func TestInstall(t *testing.T) {
	t.Parallel()

	images := []string{
		"debian-buster",
		"debian-stretch",
		"ubuntu-bionic",
		"ubuntu-xenial",
	}
	for _, img := range images {
		t.Run(img, func(t *testing.T) {
			testImage(t, img)
		})
	}
}

func testImage(t *testing.T, image string) {
	assert := assert.New(t)
	failIf := failIfFunc(t)

	buildOutput, err := sh.Output("docker", "build",
		"-f", "Dockerfile."+image,
		"-t", "testinstall_"+image,
		".",
	)
	failIf(err)

	var imageId string
	scanner := bufio.NewScanner(strings.NewReader(buildOutput))
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		if strings.HasPrefix(line, "Successfully built") {
			fields := strings.Fields(line)
			imageId = fields[len(fields)-1]
		}
	}
	failIf(scanner.Err())
	assert.NotEmpty(imageId)

	runOutput, err := sh.Output("docker", "run",
		"-d", "--privileged",
		"-v", "/sys/fs/cgroup:/sys/fs/cgroup:ro",
		imageId,
	)
	failIf(err)

	var containerId string
	scanner = bufio.NewScanner(strings.NewReader(runOutput))
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		containerId = strings.TrimSpace(line)[0:12]
	}
	defer func() {
		_ = sh.RunV("docker", "kill", containerId)
	}()
	failIf(scanner.Err())
	assert.NotEmpty(containerId)

	err = sh.RunV("docker", "exec", containerId, "curl", "-o", installFile, installURL)
	failIf(err)
	err = sh.RunV("docker", "exec", containerId, "bash", installFile)
	failIf(err)

	assert.Eventually(func() bool {
		return tequilaIsHealthy(t, containerId)
	}, 5*time.Second, 500*time.Millisecond)
}

func tequilaIsHealthy(t *testing.T, containerId string) bool {
	var healthcheckOutput struct {
		Uptime string `json:"uptime"`
	}
	healthcheckJSON, err := sh.Output("docker", "exec", containerId, "curl", "-s", "localhost:4050/healthcheck")
	if err != nil {
		return false
	}
	fmt.Println(healthcheckJSON)
	err = json.Unmarshal([]byte(healthcheckJSON), &healthcheckOutput)
	failIfFunc(t)(err)
	assert.NotEmpty(t, healthcheckOutput.Uptime)
	return true
}

func failIfFunc(t *testing.T) func(error) {
	return func(err error) {
		if err != nil {
			assert.FailNow(t, "Fatal error occurred: "+err.Error())
		}
	}
}
