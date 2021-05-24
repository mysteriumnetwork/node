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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/user"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	supervisorclient "github.com/mysteriumnetwork/node/supervisor/client"
	"github.com/mysteriumnetwork/node/utils"
)

type client struct {
	mu    sync.Mutex
	iface string
}

// New create new remote WireGuard client which communicates with supervisor.
func New() (*client, error) {
	log.Debug().Msg("Creating remote wg client")
	return &client{}, nil
}

func (c *client) ReConfigureDevice(config wgcfg.DeviceConfig) error {
	// TODO add reconnect support
	return fmt.Errorf("not supported")
}

func (c *client) ConfigureDevice(config wgcfg.DeviceConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.iface = config.IfaceName
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("could not get current OS user: %w", err)
	}

	jsonCfg, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("could not marshal device config to JSON: %w", err)
	}

	// Convert config to base64 to prevent nasty parsing issues on supervisor.
	jsonb64 := base64.StdEncoding.EncodeToString(jsonCfg)

	actualIface, err := supervisorclient.Command("wg-up", "-uid", currentUser.Uid, "-config", jsonb64)
	if err != nil {
		return fmt.Errorf("failed to create wg interface: %w", err)
	}
	log.Debug().Msgf("Tunnel interface created: %s", actualIface)
	return nil
}

func (c *client) DestroyDevice(iface string) error {
	_, err := supervisorclient.Command("wg-down", "-iface", iface)
	if err != nil {
		return fmt.Errorf("failed to destroy wg interface: %w", err)
	}
	return nil
}

func (c *client) PeerStats(iface string) (*wgcfg.Stats, error) {
	statsJSON, err := supervisorclient.Command("wg-stats", "-iface", iface)
	if err != nil {
		return nil, fmt.Errorf("failed to get wg stats: %w", err)
	}

	stats := wgcfg.Stats{}
	if err := json.Unmarshal([]byte(statsJSON), &stats); err != nil {
		return nil, fmt.Errorf("could not unmarshal stats: %w", err)
	}
	return &stats, nil
}

func (c *client) Close() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	errs := utils.ErrorCollection{}
	if err := c.DestroyDevice(c.iface); err != nil {
		errs.Add(err)
	}
	if err := errs.Error(); err != nil {
		return fmt.Errorf("could not close client: %w", err)
	}
	return nil
}
