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

package localnet

import (
	"time"

	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const composeFile = "./docker-compose.localnet.yml"

// LocalnetUp starts local environment
func LocalnetUp() error {
	logconfig.Bootstrap()
	runner := newRunner(composeFile)
	if err := runner.Up(); err != nil {
		log.Err(err).Msg("Failed to start local environment")
		if err := runner.Down(); err != nil {
			log.Err(err).Msg("Failed to cleanup environment")
		}
		return err
	}
	return nil
}

// LocalnetDown stops local environment
func LocalnetDown() error {
	logconfig.Bootstrap()
	runner := newRunner(composeFile)
	return runner.Down()
}

// NewRunner returns e2e test runners instance
func newRunner(composeFiles ...string) *Runner {
	fileArgs := make([]string, 0)
	for _, f := range composeFiles {
		fileArgs = append(fileArgs, "-f", f)
	}
	envName := "localnet"
	var args []string
	args = append(args, fileArgs...)
	args = append(args, "-p", envName)

	return &Runner{
		compose: sh.RunCmd("docker-compose", args...),
		envName: envName,
	}
}

// Runner is e2e tests runner responsible for starting test environment and running e2e tests.
type Runner struct {
	compose func(args ...string) error
	envName string
}

// Init initialises containers.
func (r *Runner) Up() error {
	if err := r.startAppContainers(); err != nil {
		return errors.Wrap(err, "could not start app containers")
	}

	if err := r.startProviderConsumerNodes(); err != nil {
		return errors.Wrap(err, "could not start provider consumer nodes")
	}
	return nil
}

func (r *Runner) Down() error {
	if err := r.compose("down", "--remove-orphans", "--timeout", "30"); err != nil {
		return errors.Wrap(err, "could not stop environment")
	}
	return nil
}

func (r *Runner) startAppContainers() error {
	log.Info().Msg("Starting other services")
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
