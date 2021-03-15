/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package netutil

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/mysteriumnetwork/node/utils/cmdutil"
)

func assignIP(iface string, subnet net.IPNet) error {
	if err := cmdutil.SudoExec("ifconfig", iface, subnet.String(), peerIP(subnet).String()); err != nil {
		return err
	}

	if err := cmdutil.SudoExec("ifconfig", iface, "inet6", "100::2"); err != nil {
		return err
	}

	return nil
}

func excludeRoute(ip, gw net.IP) error {
	return cmdutil.SudoExec("route", "add", "-host", ip.String(), gw.String())
}

func deleteRoute(ip, gw string) error {
	return cmdutil.SudoExec("route", "delete", ip, gw)
}

func addDefaultRoute(iface string) error {
	if err := cmdutil.SudoExec("route", "add", "-net", "0.0.0.0/1", "-interface", iface); err != nil {
		return err
	}

	if err := cmdutil.SudoExec("route", "add", "-net", "128.0.0.0/1", "-interface", iface); err != nil {
		return err
	}

	if err := cmdutil.SudoExec("route", "add", "-inet6", "::/1", fmt.Sprintf("100::1%%%s", iface)); err != nil {
		return err
	}

	if err := cmdutil.SudoExec("route", "add", "-inet6", "8000::/1", fmt.Sprintf("100::1%%%s", iface)); err != nil {
		return err
	}

	return nil
}

func peerIP(subnet net.IPNet) net.IP {
	lastOctetID := len(subnet.IP) - 1
	if subnet.IP[lastOctetID] == byte(1) {
		subnet.IP[lastOctetID] = byte(2)
	} else {
		subnet.IP[lastOctetID] = byte(1)
	}
	return subnet.IP
}

func logNetworkStats() {
	for _, args := range [][]string{{"ifconfig", "-a"}, {"netstat", "-rn"}, {"pfctl", "-s", "all"}} {
		out, err := exec.Command("sudo", args...).CombinedOutput()
		logOutputToTrace(out, err, args...)
	}
}
