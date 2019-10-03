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

package shaper

import (
	"net"
	"os/exec"
	"strconv"

	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/services/shared"
	"github.com/pkg/errors"
)

var log = logconfig.NewLogger()

const (
	limitKbps = 5000
)

// noopShaper does not apply to network interface
type noopShaper struct {
}

func newNoopShaper() *noopShaper {
	return &noopShaper{}
}

// TargetInterface noop
func (noopShaper) TargetInterface(interfaceName string) {
	_ = log.Errorf("target interface %s: noop", interfaceName)
}

// Apply noop
func (noopShaper) Apply() error {
	return errors.New("shaper is noop: apply will not take effect")
}

// cmd shell command to be executed with args
type cmd func(args ...string) error

// wonderShaper uses wondershaper utility to apply bandwidth limit to the network interface
type wonderShaper struct {
	runCmd          cmd
	targetInterface string
}

func newWonderShaper() (*wonderShaper, error) {
	path, err := healthcheck("wondershaper")
	if err != nil {
		return nil, errors.Wrap(err, "wondershaper healthcheck failed")
	}
	return &wonderShaper{runCmd: sh.RunCmd(path)}, nil
}

// healthcheck checks wondershaper status on the first network interface:
// this works as a healthcheck considering that a compatible version of wondershaper is installed
// If healthcheck passes, it returns an actual path of wondershaper binary.
func healthcheck(cmd string) (path string, err error) {
	log.Debug("Wondershaper healthcheck: starting")

	path, err = exec.LookPath(cmd)
	if err != nil {
		return path, errors.Wrap(err, "wondershaper executable not found")
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return path, errors.Wrap(err, "failed to lookup network interfaces")
	}
	testInterface := interfaces[0].Name

	wondershaperOut, err := sh.Output(path, testInterface)
	if err != nil {
		return path, errors.Wrapf(err, "failed to invoke wondershaper on %s", testInterface)
	}

	log.Debugf("Wondershaper healthcheck success. Status on %s: %s", testInterface, wondershaperOut)
	return path, nil
}

// TargetInterface targets specific network interface
func (s *wonderShaper) TargetInterface(interfaceName string) {
	s.targetInterface = interfaceName
}

// Apply applies current shaping rules
func (s *wonderShaper) Apply() error {
	enabled := config.Current.GetBool(shared.ShaperEnabledFlag.Name)
	if enabled {
		log.Info("Shaper enabled, limiting bandwidth")
		return s.limitBandwidth()
	}
	log.Info("Shaper disabled, removing bandwidth limit")
	return s.unlimitBandwidth()
}

func (s *wonderShaper) limitBandwidth() error {
	iface := "tun0"
	err := s.runCmd(iface, strconv.Itoa(limitKbps), strconv.Itoa(limitKbps))
	return errors.Wrap(err, "could not limit bandwidth on "+iface)
}

func (s *wonderShaper) unlimitBandwidth() error {
	iface := "tun0"
	err := s.runCmd("clear", iface)
	return errors.Wrap(err, "could not unlimit bandwidth on "+iface)
}
