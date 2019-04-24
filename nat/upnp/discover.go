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

	log "github.com/cihub/seelog"
	"github.com/huin/goupnp"
	"github.com/huin/goupnp/httpu"
	"github.com/huin/goupnp/ssdp"
	"github.com/pkg/errors"
)

const logPrefix = "[upnp]"

type gatewayDevice struct {
	root   *goupnp.RootDevice
	server string
}

// String returns human-readable string representation of gatewayDevice
func (device gatewayDevice) String() string {
	return fmt.Sprintf("%v", map[string]string{
		"server":       device.server,
		"deviceType":   device.root.Device.DeviceType,
		"manufacturer": device.root.Device.Manufacturer,
		"friendlyName": device.root.Device.FriendlyName,
		"modelName":    device.root.Device.ModelName,
		"modelNo":      device.root.Device.ModelNumber,
	})
}

// ReportNetworkGateways scans network for internet gateways supporting UPnP and reports them to stdout
func ReportNetworkGateways() {
	log.Trace(logPrefix, " Scanning for UPnP gateways")
	gateways, err := discoverGateways()
	if err != nil {
		_ = log.Error(logPrefix, errors.Wrap(err, "error discovering UPnP devices"))
		return
	}
	log.Infof("%s UPnP gateways detected: %d", logPrefix, len(gateways))
	for _, device := range gateways {
		log.Infof("%s UPnP gateway detected %v", logPrefix, device)
	}
}

func discoverGateways() ([]gatewayDevice, error) {
	client, err := httpu.NewHTTPUClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	responses, err := ssdp.SSDPRawSearch(client, ssdp.UPNPRootDevice, 2, 3)
	if err != nil {
		return nil, err
	}

	var results []gatewayDevice

	for _, response := range responses {
		device, err := parseDiscoveryResponse(response)
		if err != nil {
			_ = log.Warnf("%s error parsing discovery response %v", logPrefix, err)
			continue
		}
		if !isGateway(device) {
			log.Debugf("%s not a gateway device: %v", logPrefix, device)
			continue
		}

		results = append(results, *device)
	}

	return results, nil
}

func parseDiscoveryResponse(res *http.Response) (*gatewayDevice, error) {
	device := &gatewayDevice{server: res.Header.Get("server")}

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

func isGateway(device *gatewayDevice) bool {
	return strings.Contains(device.root.Device.DeviceType, "InternetGatewayDevice")
}
