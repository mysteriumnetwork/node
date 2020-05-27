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

package remoteclient

import (
	"fmt"
	"net"
	"os/user"

	supervisorclient "github.com/mysteriumnetwork/node/supervisor/client"
	"github.com/rs/zerolog/log"
)

// createTUN creates native TUN device for wireguard.
func createTUN(iface string, subnet net.IPNet) error {
	currentUser, err := user.Current()
	if err != nil {
		return err
	}

	actualIface, err := supervisorclient.Command("wg-up", "-iface", iface, "-uid", currentUser.Uid)
	if err != nil {
		return fmt.Errorf("failed to create wg interface: %w", err)
	}
	log.Debug().Msgf("Tunnel interface created: %s", actualIface)
	if err := assignIP(iface, subnet); err != nil {
		return fmt.Errorf("failed to assign IP address: %w", err)
	}
	return nil
}

func destroyTUN(iface string) error {
	_, err := supervisorclient.Command("wg-down", "-iface", iface)
	if err != nil {
		return fmt.Errorf("failed to destroy wg interface: %w", err)
	}
	return nil
}

func assignIP(iface string, subnet net.IPNet) error {
	_, err := supervisorclient.Command("assign-ip", "-iface", iface, "-net", subnet.String())
	return err
}
