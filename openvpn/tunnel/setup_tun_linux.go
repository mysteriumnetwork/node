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

package tunnel

import (
	"os"
	"os/exec"
	"strconv"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/openvpn/config"
)

// GetTunnelSetup gets the appropriate Setup for tunnel for linux
func GetTunnelSetup(configuration *config.GenericConfig) Setup {
	return &LinuxTunDeviceManager{}
}

// LinuxTunDeviceManager represents the tun manager for linux
type LinuxTunDeviceManager struct {
	scriptSetup string

	// runtime variables
	device tunDevice
}

// tunDevice represents tun device structure
type tunDevice struct {
	Name string
}

// Setup sets the tunel up
func (service *LinuxTunDeviceManager) Setup(configuration *config.GenericConfig) error {
	configuration.SetScriptParam("iproute", config.SimplePath("nonpriv-ip"))
	service.scriptSetup = configuration.GetFullScriptPath(config.SimplePath("prepare-env.sh"))

	device, err := findFreeTunDevice()
	if err != nil {
		return err
	}

	if err := service.createTunDevice(device); err != nil {
		return err
	}

	service.device = device
	configuration.SetPersistTun()
	configuration.SetDevice(device.Name)
	return nil
}

// Stop stops the tunnel
func (service *LinuxTunDeviceManager) Stop() {
	var err error
	var exists bool

	if exists, err = service.deviceExists(service.device); err != nil {
		log.Info(tunLogPrefix, err)
	}

	if exists {
		service.deleteDevice(service.device)
	}
}

func (service *LinuxTunDeviceManager) createTunDevice(device tunDevice) (err error) {
	err = service.createDeviceNode()
	if err != nil {
		return err
	}

	exists, err := service.deviceExists(device)
	if err != nil {
		return err
	}

	if exists {
		log.Info(tunLogPrefix, device.Name+" device already exists, attempting to use it")
		return
	}

	cmd := exec.Command("sudo", "ip", "tuntap", "add", "dev", device.Name, "mode", "tun")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed to add tun device: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
		// we should not proceed without tun device
		return err
	}

	log.Info(tunLogPrefix, device.Name+" device created")
	return nil
}

func (service *LinuxTunDeviceManager) deviceExists(device tunDevice) (exists bool, err error) {
	if _, err := os.Stat("/sys/class/net/" + device.Name); err == nil {
		return true, nil
	}

	return false, err
}

func (service *LinuxTunDeviceManager) deleteDevice(device tunDevice) {
	cmd := exec.Command("sudo", "ip", "tuntap", "delete", "dev", device.Name, "mode", "tun")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed to remove tun device: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
	} else {
		log.Info(tunLogPrefix, device.Name, " device removed")
	}
}

// FindFreeTunDevice returns first free tun device on system
func findFreeTunDevice() (tun tunDevice, err error) {
	// search only among first 10 tun devices
	for i := 0; i <= 10; i++ {
		tunName := "tun" + strconv.Itoa(i)
		tunFile := "/sys/class/net/tun" + tunName
		if _, err := os.Stat(tunFile); os.IsNotExist(err) {
			return tunDevice{tunName}, nil
		}
	}

	return tun, ErrNoFreeTunDevice
}

func (service *LinuxTunDeviceManager) createDeviceNode() error {
	if _, err := os.Stat("/dev/net/tun"); err == nil {
		// device node already exists
		return nil
	}

	cmd := exec.Command("sudo", service.scriptSetup)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed to execute tun script: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
		return err
	}

	log.Info(tunLogPrefix, "/dev/net/tun device node created")
	return nil
}
