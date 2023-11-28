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

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// NewRunner returns e2e test runners instance
func NewRunner(composeFiles []string, testEnv, services string) (runner *Runner, cleanup func()) {
	fileArgs := make([]string, 0)
	for _, f := range composeFiles {
		fileArgs = append(fileArgs, "-f", f)
	}
	var args []string
	args = append(args, fileArgs...)
	args = append(args, "-p", testEnv)

	runner = &Runner{
		compose:    sh.RunCmd("docker-compose", args...),
		composeOut: sh.OutCmd("docker-compose", args...),
		testEnv:    testEnv,
		services:   services,
	}
	return runner, runner.cleanup
}

// Runner is e2e tests runner responsible for starting test environment and running e2e tests.
type Runner struct {
	compose         func(args ...string) error
	composeOut      func(args ...string) (string, error)
	etherPassphrase string
	testEnv         string
	services        string
}

// Test starts given provider and consumer nodes and runs e2e tests.
func (r *Runner) Test(providerHost string) (retErr error) {
	services := strings.Split(r.services, ",")
	if err := r.startProviderConsumerNodes(providerHost, services); err != nil {
		retErr = errors.Wrap(err, "tests failed!")
		return
	}

	defer func() {
		if err := r.stopProviderConsumerNodes(providerHost, services); err != nil {
			log.Err(err).Msg("Could not stop provider consumer nodes")
		}

		if retErr == nil { // check public IPs in logs only if all the tests succeeded
			if err := r.checkPublicIPInLogs("myst-provider", "myst-consumer-wireguard"); err != nil {
				retErr = errors.Wrap(err, "tests failed!")
				return
			}
		}
	}()

	log.Info().Msg("Running tests for env: " + r.testEnv)

	err := r.compose("run", "go-runner",
		"/usr/local/bin/test", "-test.v",
		"-provider.tequilapi-host", providerHost,
		"-provider.tequilapi-port=4050",
		"-consumer.tequilapi-port=4050",
		"-consumer.services", r.services,
	)

	retErr = errors.Wrap(err, "tests failed!")
	return
}

func (r *Runner) checkPublicIPInLogs(containers ...string) error {
	regExps := []*regexp.Regexp{
		regexp.MustCompile(`(^|[^0-9])(172\.30\.0\.2)($|[^0-9])`),
		regexp.MustCompile(`(^|[^0-9])(172\.31\.0\.2)($|[^0-9])`),
	}

	for _, containerName := range containers {
		output, err := r.composeOut("logs", containerName)
		if err != nil {
			log.Err(err).Msgf("Could not get logs of %s container", containerName)
			continue
		}

		if len(output) == 0 {
			log.Error().Msgf("Could not get logs of %s container. Empty data", containerName)
			continue
		}

		for _, reg := range regExps {
			if reg.MatchString(output) {
				// it will be easier to locate the place if we print the output
				log.Warn().Msgf("output from %s container's logs:\n%s", containerName, output)
				return fmt.Errorf("found public IP address by regular expression %s in %s container's logs", reg.String(), containerName)
			}
		}
	}

	return nil
}

func (r *Runner) cleanup() {
	log.Info().Msg("Cleaning up")

	_ = r.compose("logs")
	if err := r.compose("down", "--volumes", "--remove-orphans", "--timeout", "30"); err != nil {
		log.Warn().Err(err).Msg("Cleanup error")
	}
}

// Init starts provider and consumer node dependencies.
func (r *Runner) Init() error {
	log.Info().Msg("Starting other services")
	if err := r.compose("pull"); err != nil {
		return errors.Wrap(err, "could not pull images")
	}

	if err := r.compose("up", "-d", "broker", "ganache", "ganache2", "ipify", "morqa", "mongodb", "transactordatabase", "pilvytis-mock"); err != nil {
		return errors.Wrap(err, "starting other services failed!")
	}

	log.Info().Msg("Starting discovery DB")
	if err := r.compose("up", "-d", "db"); err != nil {
		return errors.Wrap(err, "starting DB failed!")
	}

	log.Info().Msg("Starting discovery")
	if err := r.compose("up", "-d", "discovery"); err != nil {
		return errors.Wrap(err, "starting mysterium-api failed!")
	}

	log.Info().Msg("Starting httpmock")
	if err := r.compose("up", "-d", "http-mock"); err != nil {
		return errors.Wrap(err, "starting http-mock failed!")
	}

	log.Info().Msg("building runner")
	if err := r.compose("build", "go-runner"); err != nil {
		return fmt.Errorf("could not build go runner %w", err)
	}

	log.Info().Msg("Deploying contracts")
	err := r.compose("run", "go-runner",
		"/usr/local/bin/deployer",
		"--keystore.directory=./keystore",
		"--ether.address=0x354Bd098B4eF8c9E70B7F21BE2d455DF559705d7",
		fmt.Sprintf("--ether.passphrase=%v", r.etherPassphrase),
		"--geth.url=http://ganache:8545")
	if err != nil {
		return errors.Wrap(err, "failed to deploy contracts!")
	}

	log.Info().Msg("Deploying contracts to bc2")
	err = r.compose("run", "go-runner",
		"/usr/local/bin/deployer",
		"--keystore.directory=./keystore",
		"--ether.address=0x354Bd098B4eF8c9E70B7F21BE2d455DF559705d7",
		fmt.Sprintf("--ether.passphrase=%v", r.etherPassphrase),
		"--geth.url=ws://ganache2:8545")
	if err != nil {
		return errors.Wrap(err, "failed to deploy contracts!")
	}

	log.Info().Msg("Seeding http mock")
	if err := seedHTTPMock(); err != nil {
		return fmt.Errorf("could not seed http mock %w", err)
	}

	log.Info().Msg("Starting transactor")
	if err := r.compose("up", "-d", "transactor", "transactor-sidecar"); err != nil {
		return errors.Wrap(err, "starting transactor failed!")
	}

	log.Info().Msg("Building app images")
	if err := r.compose("build"); err != nil {
		return errors.Wrap(err, "building app images failed!")
	}

	return nil
}

func (r *Runner) startProviderConsumerNodes(providerHost string, services []string) error {
	log.Info().Msg("Starting provider consumer containers")

	args := []string{
		"up",
		"-d",
		providerHost,
	}

	for i := range services {
		args = append(args, fmt.Sprintf("myst-consumer-%v", services[i]))
	}

	if err := r.compose(args...); err != nil {
		return errors.Wrap(err, "starting app containers failed!")
	}
	return nil
}

func (r *Runner) stopProviderConsumerNodes(providerHost string, services []string) error {
	log.Info().Msg("Stopping provider consumer containers")

	args := []string{
		"stop",
		providerHost,
	}

	for i := range services {
		args = append(args, fmt.Sprintf("myst-consumer-%v", services[i]))
	}

	if err := r.compose(args...); err != nil {
		return errors.Wrap(err, "stopping containers failed!")
	}
	return nil
}

func seedHTTPMock() error {
	url := "http://localhost:9999/expectation"
	method := "PUT"

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	for _, v := range httpMockExpectations {
		marshaled, err := json.Marshal(v)
		if err != nil {
			return err
		}

		req, err := http.NewRequest(method, url, bytes.NewReader(marshaled))
		if err != nil {
			return err
		}

		req.Header.Add("Content-Type", "application/json")
		_, err = client.Do(req)
		if err != nil {
			return err
		}
	}

	return nil
}

var httpMockExpectations = []HTTPMockExpectation{
	{
		HTTPRequest: HTTPRequest{
			Method: "GET",
			Path:   "/gecko/simple/price",
		},
		HTTPResponse: HTTPResponse{
			StatusCode: http.StatusOK,
			Headers: []Headers{
				{
					Name:   "Content-Type",
					Values: []string{"application/json"},
				},
			},
			Body: `{"mysterium":{"usd":1,"eur":1},"matic-network":{"usd":0.5,"eur":0.5},"ethereum":{"usd":0.00001,"eur":0.00001}}`,
		},
	},
}

// HTTPMockExpectation the expectation payload.
type HTTPMockExpectation struct {
	HTTPRequest  HTTPRequest  `json:"httpRequest"`
	HTTPResponse HTTPResponse `json:"httpResponse"`
}

// HTTPRequest the http request properties for http mock.
type HTTPRequest struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Headers the http response headers for http mock.
type Headers struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// HTTPResponse the http response for http mock.
type HTTPResponse struct {
	StatusCode int       `json:"statusCode"`
	Headers    []Headers `json:"headers"`
	Body       string    `json:"body"`
}
