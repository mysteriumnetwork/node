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

package mmn

import (
	"net"
	"os/exec"
	"runtime"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/metadata"
)

// NodeInformation contains node information to be sent to MMN
type NodeInformation struct {
	MACAddress  string `json:"mac_address"`
	LocalIP     string `json:"local_ip"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	NodeVersion string `json:"node_version"`
	Identity    string `json:"identity"`
}

// OnIdentityUnlockCallback sends node information to MMN on identity unlock
func OnIdentityUnlockCallback(client *client) func(string) {
	return func(identity string) {
		info, err := getNodeInformation()
		if err != nil {
			log.Error(errors.Wrap(err, "failed to get NodeInformation for MMN"))
			return
		}
		info.Identity = identity
		if err = client.RegisterNode(info); err != nil {
			log.Error(errors.Wrap(err, "failed to send NodeInformation to MMN"))
		}
	}
}

func getNodeInformation() (*NodeInformation, error) {
	var mac, ip string
	ip, err := getLocalNetworkIP()

	if err != nil {
		return nil, errors.Wrap(err, "failed to get local network IP")
	}

	mac, err = getMACAddress(ip)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get MAC address")
	}

	info := &NodeInformation{
		MACAddress:  mac,
		LocalIP:     ip,
		Arch:        runtime.GOOS + "/" + runtime.GOARCH,
		OS:          getOS(),
		NodeVersion: metadata.VersionAsString(),
	}

	return info, nil
}

func getOS() string {
	if output := getOSByCommand("darwin", "sw_vers", "-productVersion"); len(output) > 0 {
		return "MAC OS X - " + strings.TrimSpace(string(output))
	}

	if output := getOSByCommand("linux", "lsb_release", "-d"); len(output) > 0 {
		return strings.TrimSpace(strings.Replace(string(output), "Description:", "", 1))
	}

	return ""
}

func getOSByCommand(os string, command string, args ...string) string {
	if runtime.GOOS == os {
		output, err := exec.Command(command, args...).Output()
		if err != nil {
			log.Error(errors.Wrap(err, "failed to get OS information for "+os+" using "+command))
			return ""
		}
		return string(output)
	}

	return ""
}

func getLocalNetworkIP() (ip string, err error) {
	addresses, err := net.InterfaceAddrs()

	if err != nil {
		log.Error("Failed to get network interface addresses", err)
		return "", err
	}

	for _, address := range addresses {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
			}
		}
	}

	return
}

func getMACAddress(ip string) (string, error) {
	var currentNetworkHardwareName string
	interfaces, _ := net.Interfaces()
	for _, i := range interfaces {

		if addresses, err := i.Addrs(); err == nil {
			for _, addr := range addresses {
				// only interested in the name with current IP address
				if strings.Contains(addr.String(), ip) {
					currentNetworkHardwareName = i.Name
				}
			}
		}
	}

	netInterface, err := net.InterfaceByName(currentNetworkHardwareName)

	if err != nil {
		log.Error("Failed to get MAC address", err)

		return "", err
	}

	macAddress, err := net.ParseMAC(netInterface.HardwareAddr.String())

	if err != nil {
		log.Error("Failed to validate MAC address", err)
		return "", err
	}

	return macAddress.String(), nil
}
