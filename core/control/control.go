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

type RuntimeServiceOptions struct {
	Name           string         `json:"name,omitempty"`
	Exec           string         `json:"exec,omitempty"`
	OCIArtifact    string         `json:"oci_artifact,omitempty"`
	ServicePort    int            `json:"service_port,omitempty"`
	ResourceLimits resourceLimits `json:"resource_limits,omitempty"`
}

type resourceLimits struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
	Disk   string `json:"disk,omitempty"`
}

// ControlPlane is a struct that represents the control plane of the node
type ControlPlane struct {
	nats           communication.Receiver
	api            *client.Client
	identity       string
	runtimeBackend runtime_service_options.Backend
}

// NewControlPlane creates a new control plane
func NewControlPlane(connection nats.Connection, api *client.Client, runtimeBackend runtime_service_options.Backend) *ControlPlane {
	return &ControlPlane{
		nats:           nats.NewReceiver(connection, communication.NewCodecJSON(), ""),
		api:            api,
		runtimeBackend: runtimeBackend,
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

// ExecuteJSON executes control-plane requests directly from JSON payload.
// It reuses the same handler logic as broker-delivered messages.
func (c *ControlPlane) ExecuteJSON(payload []byte) error {
	var request controlMessage
	if err := json.Unmarshal(payload, &request); err != nil {
		return err
	}

	if c.identity == "" {
		c.detectIdentityFromServices()
	}

	return c.handler(request)
}

func (c *ControlPlane) detectIdentityFromServices() {
	services, err := c.api.Services()
	if err != nil {
		return
	}

	for _, svc := range services {
		if svc.ProviderID != "" {
			c.identity = svc.ProviderID
			return
		}
	}
}
