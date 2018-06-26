/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package linux

import (
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/utils"
	"os"
)

const tunLogPrefix = "[linux tun service] "

type serviceLinuxTun struct {
	device TunnelDevice
}

// NewLinuxTunnelService creates linux specific tunnel manager for interface creation and removal
func NewLinuxTunnelService() TunnelService {
	return &serviceLinuxTun{}
}

func (service *serviceLinuxTun) Add(device TunnelDevice) {
	service.device = device
}

func (service *serviceLinuxTun) Start() error {
	return service.createTunDevice()
}

func (service *serviceLinuxTun) Stop() {
	var err error
	var exists bool

	if exists, err = service.deviceExists(); err != nil {
		log.Info(tunLogPrefix, err)
	}

	if exists {
		service.deleteDevice()
	}
}

func (service *serviceLinuxTun) createTunDevice() (err error) {
	exists, err := service.deviceExists()
	if err != nil {
		return err
	}

	if exists {
		log.Info(tunLogPrefix, service.device.Name+" device already exists, attempting to use it")
		return
	}

	cmd := utils.SplitCommand("sudo", "ip tuntap add dev "+service.device.Name+" mode tun")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed to add tun device: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
		// we should not proceed without tun device
		return err
	}

	log.Info(tunLogPrefix, service.device.Name+" device created")
	return nil
}

func (service *serviceLinuxTun) deviceExists() (exists bool, err error) {
	if _, err := os.Stat("/sys/class/net/" + service.device.Name); err == nil {
		return true, nil
	}

	return false, err
}

func (service *serviceLinuxTun) deleteDevice() {
	cmd := utils.SplitCommand("sudo", "ip tuntap delete dev "+service.device.Name+" mode tun")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed to remove tun device: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
	} else {
		log.Info(tunLogPrefix, service.device.Name, " device removed")
	}
}
