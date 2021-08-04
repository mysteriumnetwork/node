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

package service

import (
	"encoding/json"
	"net"

	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/nat"
	natevent "github.com/mysteriumnetwork/node/nat/event"
)

// NATEventGetter allows us to fetch the last known NAT event
type NATEventGetter interface {
	LastEvent() *natevent.Event
}

// NewManager creates new instance of Wireguard service
func NewManager(
	ipResolver ip.Resolver,
	country string,
	natService nat.NATService,
	eventPublisher eventbus.Publisher,
	options Options,
	portSupplier port.ServicePortSupplier,
	trafficFirewall firewall.IncomingTrafficFirewall,
) *Manager {
	return &Manager{}
}

// Manager represents an instance of Wireguard service
type Manager struct{}

// ProvideConfig provides the config for consumer
func (manager *Manager) ProvideConfig(_ string, _ json.RawMessage, _ *net.UDPConn) (*service.ConfigParams, error) {
	return nil, errors.New("not implemented")
}

// Serve starts service - does block
func (manager *Manager) Serve(_ *service.Instance) error {
	return errors.New("not implemented")
}

// Stop stops service.
func (manager *Manager) Stop() error {
	return errors.New("not implemented")
}
