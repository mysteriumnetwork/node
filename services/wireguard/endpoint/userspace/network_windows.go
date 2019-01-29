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

package userspace

import (
	"net"
	"os/exec"
	"strconv"

	"github.com/jackpal/gateway"
	"github.com/pkg/errors"
)

func assignIP(iface string, subnet net.IPNet) error {
	out, err := exec.Command("powershell", "-Command", "netsh interface ip set address name=\""+iface+"\" source=static "+subnet.String()).CombinedOutput()
	return errors.Wrap(err, string(out))
}

func renameInterface(name, newname string) error {
	out, err := exec.Command("powershell", "-Command", "netsh interface set interface name=\""+name+"\" newname=\""+newname+"\"").CombinedOutput()
	return errors.Wrap(err, string(out))
}

func excludeRoute(ip net.IP) error {
	gw, err := gateway.DiscoverGateway()
	if err != nil {
		return err
	}

	out, err := exec.Command("powershell", "-Command", "route add "+ip.String()+"/32 "+gw.String()).CombinedOutput()
	return errors.Wrap(err, string(out))
}

func addDefaultRoute(name string) error {
	id, gw, err := interfaceInfo(name)
	if err != nil {
		return errors.Wrap(err, "failed to get info of interface: " + name)
	}

	if out, err := exec.Command("powershell", "-Command", "route add 0.0.0.0/1 "+gw+" if "+id).CombinedOutput(); err != nil {
		return errors.Wrap(err, string(out))
	}

	out, err := exec.Command("powershell", "-Command", "route add 128.0.0.0/1 "+gw+" if "+id).CombinedOutput()
	return errors.Wrap(err, string(out))
}

func destroyDevice(name string) error {
	return nil
}

func interfaceInfo(name string) (id, gw string, err error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get interfaces "+name)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get interfaces addresses")
	}

	var ipv4 net.IP
	for _, addr := range addrs {
		ip, _, _ := net.ParseCIDR(addr.String())
		if ip.To4() == nil {
			continue
		}

		if ipv4.Equal(net.IPv4zero) {
			return "", "", errors.New("failed to get interface info: exactly 1 IPv4 expected")
		}

		ipv4 = ip.To4()
		ipv4[net.IPv4len-1] = byte(1)
	}

	return strconv.Itoa(iface.Index), ipv4.String(), nil
}
