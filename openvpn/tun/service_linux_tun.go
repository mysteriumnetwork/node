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

package tun

import (
	"fmt"
	"os/exec"

	"bytes"
	log "github.com/cihub/seelog"
	"os"
)

const tunLogPrefix = "[tun] "

type serviceLinuxTun struct {
	device Device
}

func (service *serviceLinuxTun) Add(device Device) {
	service.device = device
}

func (service *serviceLinuxTun) Start() error {
	return service.createTunDevice()
}

func (service *serviceLinuxTun) Stop() (err error) {
	var exists bool

	if exists, err = service.deviceExists(); err != nil {
		return err
	}

	if !exists {
		return nil
	}

	return service.deleteDevice()
}

func (service *serviceLinuxTun) createTunDevice() (err error) {

	if exists, err := service.deviceExists(); exists {
		log.Info(tunLogPrefix, service.device.Name+" device already exists, attempting to use it")
		return err
	}

	var stderr bytes.Buffer
	cmd := exec.Command(
		"bash",
		"-c",
		"sudo ip tuntap add dev "+service.device.Name+" mode tun",
	)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to add tun device: %s: %s", err, stderr.String())
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

func (service *serviceLinuxTun) deleteDevice() (err error) {
	var stderr bytes.Buffer
	cmd := exec.Command(
		"bash",
		"-c",
		"sudo ip tuntap delete dev "+service.device.Name+" mode tun",
	)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to remove tun device: %s: %s", err, stderr.String())
	}

	log.Info(tunLogPrefix, service.device.Name+" device removed")
	return nil
}
