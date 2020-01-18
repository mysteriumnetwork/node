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
	"fmt"
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
		compose:  sh.RunCmd("docker-compose", args...),
		testEnv:  testEnv,
		services: services,
	}
	return runner, runner.cleanup
}

// Runner is e2e tests runner responsible for starting test environment and running e2e tests.
type Runner struct {
	compose         func(args ...string) error
	etherPassphrase string
	testEnv         string
	services        string
}

// Init initialises containers.
func (r *Runner) Init() error {
	if err := r.startAppContainers(); err != nil {
		return err
	}

	if err := r.startProviderConsumerNodes(); err != nil {
		return err
	}
	return nil
}

// Test runs e2e tests.
func (r *Runner) Test() error {
	log.Info().Msg("Running tests for env: " + r.testEnv)

	err := r.compose("run", "go-runner",
		"go", "test", "-v", "./e2e/...", "-args",
		"--deployer.keystore-directory=../e2e/blockchain/keystore",
		"--deployer.address=0x354Bd098B4eF8c9E70B7F21BE2d455DF559705d7",
		"--deployer.passphrase", r.etherPassphrase,
		"--provider.tequilapi-host=myst-provider",
		"--provider.tequilapi-port=4050",
		"--consumer.tequilapi-host=myst-consumer",
		"--consumer.tequilapi-port=4050",
		"--geth.url=ws://ganache:8545",
		"--consumer.services", r.services,
	)
	return errors.Wrap(err, "tests failed!")
}

func (r *Runner) cleanup() {
	log.Info().Msg("Cleaning up")
	_ = r.compose("logs")
	if err := r.compose("down", "--volumes", "--remove-orphans", "--timeout", "30"); err != nil {
		log.Warn().Err(err).Msg("Cleanup error")
	}
}

func (r *Runner) startAppContainers() error {
	log.Info().Msg("Starting other services")
	if err := r.compose("pull"); err != nil {
		return errors.Wrap(err, "could not pull images")
	}
	if err := r.compose("up", "-d", "broker", "ganache", "ipify", "morqa"); err != nil {
		return errors.Wrap(err, "starting other services failed!")
	}
	log.Info().Msg("Starting DB")
	if err := r.compose("up", "-d", "db"); err != nil {
		return errors.Wrap(err, "starting DB failed!")
	}

	dbUp := false
	for start := time.Now(); !dbUp && time.Since(start) < 60*time.Second; {
		err := r.compose("exec", "-T", "db", "mysqladmin", "ping", "--protocol=TCP", "--silent")
		if err != nil {
			log.Info().Msg("Waiting...")
		} else {
			log.Info().Msg("DB is up")
			dbUp = true
			break
		}
	}
	if !dbUp {
		return errors.New("starting DB timed out")
	}

	log.Info().Msg("Starting transactor")
	if err := r.compose("up", "-d", "transactor"); err != nil {
		return errors.Wrap(err, "starting transactor failed!")
	}

	log.Info().Msg("Migrating DB")
	if err := r.compose("run", "--entrypoint", "bin/db-upgrade", "mysterium-api"); err != nil {
		return errors.Wrap(err, "migrating DB failed!")
	}

	log.Info().Msg("Starting mysterium-api")
	if err := r.compose("up", "-d", "mysterium-api"); err != nil {
		return errors.Wrap(err, "starting mysterium-api failed!")
	}

	log.Info().Msg("Deploying contracts")
	err := r.compose("run", "go-runner",
		"go", "run", "./e2e/blockchain/deployer.go",
		"--keystore.directory=./e2e/blockchain/keystore",
		"--ether.address=0x354Bd098B4eF8c9E70B7F21BE2d455DF559705d7",
		fmt.Sprintf("--ether.passphrase=%v", r.etherPassphrase),
		"--geth.url=ws://ganache:8545")
	if err != nil {
		return errors.Wrap(err, "failed to deploy contracts!")
	}

	log.Info().Msg("starting accountant")
	if err := r.compose("up", "-d", "accountant"); err != nil {
		return errors.Wrap(err, "starting accountant failed!")
	}

	return nil
}

func (r *Runner) startProviderConsumerNodes() error {
	log.Info().Msg("Building app images")
	if err := r.compose("build"); err != nil {
		return errors.Wrap(err, "building app images failed!")
	}

	log.Info().Msg("Starting app containers")
	if err := r.compose("up", "-d", "myst-provider", "myst-consumer"); err != nil {
		return errors.Wrap(err, "starting app containers failed!")
	}
	return nil
}
