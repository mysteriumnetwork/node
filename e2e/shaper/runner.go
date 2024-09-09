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

package shaper

import (
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
		compose:    sh.RunCmd("docker", append([]string{"compose"}, args...)...),
		composeOut: sh.OutCmd("docker", append([]string{"compose"}, args...)...),
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
func (r *Runner) Test() (retErr error) {
	log.Info().Msg("Running tests for env: " + r.testEnv)

	err := r.compose("run", "go-runner",
		"/usr/local/bin/shaper.test",
	)

	retErr = errors.Wrap(err, "tests failed!")
	return
}

func (r *Runner) cleanup() {
	log.Info().Msg("Cleaning up")

	_ = r.compose("logs")
	if err := r.compose("down", "--volumes", "--remove-orphans", "--timeout", "30"); err != nil {
		log.Warn().Err(err).Msg("Cleanup error")
	}
}

// Init starts bug6022.test dependency
func (r *Runner) Init() error {
	log.Info().Msg("Starting other services")
	if err := r.compose("pull"); err != nil {
		return errors.Wrap(err, "could not pull images")
	}

	if err := r.compose("up", "-d", "shaper-websvc"); err != nil {
		return errors.Wrap(err, "starting other services failed!")
	}

	log.Info().Msg("Building app images")
	if err := r.compose("build"); err != nil {
		return errors.Wrap(err, "building app images failed!")
	}

	return nil
}
