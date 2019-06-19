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

package upnp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mysteriumnetwork/node/firewall"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/httpu"
	"github.com/huin/goupnp/ssdp"
	"github.com/pkg/errors"
)

// GatewayDevice represents a scanned gateway device
type GatewayDevice struct {
	root   *goupnp.RootDevice
	server string
}

// ToMap returns a map representation of the device
func (device GatewayDevice) ToMap() map[string]string {
	return map[string]string{
		"server":       device.server,
		"deviceType":   device.root.Device.DeviceType,
		"manufacturer": device.root.Device.Manufacturer,
		"friendlyName": device.root.Device.FriendlyName,
		"modelName":    device.root.Device.ModelName,
		"modelNo":      device.root.Device.ModelNumber,
	}
}

// String returns human-readable string representation of gatewayDevice
func (device GatewayDevice) String() string {
	return fmt.Sprintf("%v", device.ToMap())
}

func printGatewayInfo(gateways []GatewayDevice) {
	log.Infof("UPnP gateways detected: %d", len(gateways))
	for _, device := range gateways {
		log.Infof("UPnP gateway detected %v", device)
	}
}

func discoverGateways() ([]GatewayDevice, error) {
	client, err := httpu.NewHTTPUClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// this IP address used to discover gateways - allow it
	if _, err := firewall.AllowIPAccess("239.255.255.250"); err != nil {
		return nil, err
	}

	responses, err := ssdp.SSDPRawSearch(client, ssdp.UPNPRootDevice, 2, 3)
	if err != nil {
		return nil, err
	}

	var results []GatewayDevice

	for _, response := range responses {
		device, err := parseDiscoveryResponse(response)
		if err != nil {
			log.Warnf("error parsing discovery response %v", err)
			continue
		}
		if !isGateway(device) {
			log.Debugf("not a gateway device: %v", device)
			continue
		}

		results = append(results, *device)
	}

	return results, nil
}

func parseDiscoveryResponse(res *http.Response) (*GatewayDevice, error) {
	device := &GatewayDevice{server: res.Header.Get("server")}

	loc, err := res.Location()
	if err != nil {
		return nil, errors.Wrap(err, "unexpected bad location from search")
	}

	device.root, err = goupnp.DeviceByURL(loc)
	if err != nil {
		return nil, errors.Wrap(err, "error querying device by location")
	}

	return device, nil
}

func isGateway(device *GatewayDevice) bool {
	return strings.Contains(device.root.Device.DeviceType, "InternetGatewayDevice")
}
