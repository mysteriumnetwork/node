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

package test

import (
	"bufio"
	"os"
	"time"

	log "github.com/cihub/seelog"
	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
)

const projectName = "node_e2e_test"

var composeFiles = []string{
	"bin/localnet/docker-compose.yml",
	"e2e/docker-compose.yml",
}

type runner struct {
	compose         func(args ...string) error
	etherPassphrase string
}

// TestE2E runs end-to-end tests
func TestE2E() error {
	logconfig.Bootstrap()
	r := newRunner()
	defer r.cleanup()

	if err := r.setup(); err != nil {
		return err
	}

	return r.test()
}

func (r *runner) setup() error {
	log.Info("initializing geth node")
	if err := r.compose("run", "geth", "init", "genesis.json"); err != nil {
		return errors.Wrap(err, "initializing geth node failed!")
	}
	log.Info("starting other services")
	if err := r.compose("up", "-d", "broker", "geth"); err != nil {
		return errors.Wrap(err, "starting other services failed!")
	}
	log.Info("starting DB")
	if err := r.compose("up", "-d", "db"); err != nil {
		return errors.Wrap(err, "starting DB failed!")
	}

	dbUp := false
	for start := time.Now(); !dbUp && time.Since(start) < 30*time.Second; {
		err := r.compose("exec", "-T", "db", "mysqladmin", "ping", "--protocol=TCP", "--silent")
		if err != nil {
			log.Info("Waiting...")
		} else {
			log.Info("DB is up")
			dbUp = true
			break
		}
	}
	if !dbUp {
		return errors.New("starting DB timed out")
	}

	log.Info("migrating DB")
	if err := r.compose("run", "--entrypoint", "bin/db-upgrade", "discovery"); err != nil {
		return errors.Wrap(err, "migrating DB failed!")
	}

	log.Info("starting discovery")
	if err := r.compose("up", "-d", "discovery"); err != nil {
		return errors.Wrap(err, "starting discovery failed!")
	}

	file, err := os.Open("bin/localnet/deployer/local_acc_password.txt")
	if err != nil {
		return errors.Wrap(err, "failed to read ether passphrase!")
	}
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return errors.New("failed to read ether passphrase!")
	}
	r.etherPassphrase = scanner.Text()

	log.Info("deploying contracts")
	err = r.compose("run", "go-runner",
		"go", "run", "bin/localnet/deployer/deployer.go",
		"--keystore.directory=bin/localnet/deployer/keystore",
		"--ether.address=0xa754f0d31411d88e46aed455fa79b9fced122497",
		"--ether.passphrase", r.etherPassphrase,
		"--geth.url=http://geth:8545")
	if err != nil {
		return errors.Wrap(err, "failed to deploy contracts!")
	}

	log.Info("building app images")
	if err := r.compose("build"); err != nil {
		return errors.Wrap(err, "building app images failed!")
	}
	log.Info("starting app containers")
	if err := r.compose("up", "-d", "myst-provider", "myst-consumer", "ipify"); err != nil {
		return errors.Wrap(err, "starting app containers failed!")
	}

	return nil
}

func (r *runner) test() error {
	log.Info("running tests")
	err := r.compose("run", "go-runner",
		"go", "test", "-v", "./e2e/...", "-args",
		"--deployer.keystore-directory=../bin/localnet/deployer/keystore",
		"--deployer.address=0xa754f0d31411d88e46aed455fa79b9fced122497",
		"--deployer.passphrase", r.etherPassphrase,
		"--provider.tequilapi-host=myst-provider",
		"--provider.tequilapi-port=4050",
		"--consumer.tequilapi-host=myst-consumer",
		"--consumer.tequilapi-port=4050",
		"--geth.url=http://geth:8545",
	)
	return errors.Wrap(err, "tests failed!")
}

func (r *runner) cleanup() {
	log.Info("cleaning up")
	_ = r.compose("logs")
	if err := r.compose("down", "--volumes", "--remove-orphans", "--timeout", "30"); err != nil {
		log.Warn("cleanup error", err)
	}
}

func newRunner() *runner {
	fileArgs := make([]string, 0)
	for _, f := range composeFiles {
		fileArgs = append(fileArgs, "-f", f)
	}
	var args []string
	args = append(args, fileArgs...)
	args = append(args, "-p", projectName)

	return &runner{
		compose: sh.RunCmd("docker-compose", args...),
	}
}
