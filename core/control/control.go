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

package control

import (
	"encoding/json"
	"fmt"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	runtime_service_options "github.com/mysteriumnetwork/node/services/runtime/service"
	"github.com/mysteriumnetwork/node/tequilapi/client"
)

type controlMessage []controlMessageItem

type controlMessageItem struct {
	Service        string          `json:"service"`
	Command        string          `json:"command"`
	ProviderID     string          `json:"provider_id,omitempty"`
	AccessPolicies []string        `json:"access_policies,omitempty"`
	Options        json.RawMessage `json:"options,omitempty"`
}

// RuntimeServiceOptions aliases the immutable create contract to prevent
// control-plane schema drift.
type RuntimeServiceOptions = runtime_service_options.CreateOptions

// ControlPlane is a struct that represents the control plane of the node
type ControlPlane struct {
	nats           communication.Receiver
	api            *client.Client
	identity       string
	runtimeBackend runtime_service_options.Backend
	// runtimeInstaller is the only way a control message can install a
	// workload: it admits nothing the corporate service registry does not list.
	runtimeInstaller runtime_service_options.Installer
}

// NewControlPlane creates a new control plane
func NewControlPlane(
	connection nats.Connection,
	api *client.Client,
	runtimeBackend runtime_service_options.Backend,
	runtimeInstaller runtime_service_options.Installer,
) *ControlPlane {
	return &ControlPlane{
		nats:             nats.NewReceiver(connection, communication.NewCodecJSON(), ""),
		api:              api,
		runtimeBackend:   runtimeBackend,
		runtimeInstaller: runtimeInstaller,
	}
}

// Start starts the control plane
func (c *ControlPlane) Start(identity string) error {
	c.identity = identity
	return c.nats.Receive(communication.MessageConsumer(&consumer{
		callback: c.handler,
		topic:    communication.MessageEndpoint(fmt.Sprintf("%s.control-plane.v1", identity)),
	}))
}

// Stop stops the control plane
func (c *ControlPlane) Stop() {
	c.nats.ReceiveUnsubscribe(communication.MessageEndpoint(fmt.Sprintf("%s.control-plane.v1", c.identity)))
}
