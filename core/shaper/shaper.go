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
	"github.com/mysteriumnetwork/node/eventbus"
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

// Start noop
func (noopShaper) Start(interfaceName string) error {
	log.Info("Noop shaper: nothing will happen to interface %s", interfaceName)
	return nil
}

// cmd shell command to be executed with args
type cmd func(args ...string) error

// wonderShaper uses wondershaper utility to apply bandwidth limit to the network interface
type wonderShaper struct {
	runCmd          cmd
	targetInterface string
	eventBus        eventbus.EventBus
}

func newWonderShaper(eventBus eventbus.EventBus) (*wonderShaper, error) {
	path, err := healthcheck("wondershaper")
	if err != nil {
		return nil, errors.Wrap(err, "wondershaper healthcheck failed")
	}
	return &wonderShaper{
		runCmd:   sh.RunCmd(path),
		eventBus: eventBus,
	}, nil
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

// Start applies current shaping configuration for the specified interface
// and then continuously ensures it by listening to configuration updates
func (s *wonderShaper) Start(interfaceName string) error {
	if interfaceName == "" {
		return errors.New("interface name is empty")
	}
	s.targetInterface = interfaceName
	err := s.eventBus.SubscribeAsync(config.Topic(shared.ShaperEnabledFlag.Name), s.apply)
	if err != nil {
		return err
	}
	return s.apply()
}

func (s *wonderShaper) apply() error {
	enabled := config.Current.GetBool(shared.ShaperEnabledFlag.Name)
	if enabled {
		log.Info("Shaper enabled, limiting bandwidth")
		return s.limitBandwidth()
	}
	log.Info("Shaper disabled, removing bandwidth limit")
	return s.unlimitBandwidth()
}

func (s *wonderShaper) limitBandwidth() error {
	err := s.runCmd(s.targetInterface, strconv.Itoa(limitKbps), strconv.Itoa(limitKbps))
	return errors.Wrap(err, "could not limit bandwidth on "+s.targetInterface)
}

func (s *wonderShaper) unlimitBandwidth() error {
	err := s.runCmd("clear", s.targetInterface)
	return errors.Wrap(err, "could not unlimit bandwidth on "+s.targetInterface)
}
