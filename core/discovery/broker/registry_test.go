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

package broker

import (
	"encoding/json"
	"testing"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

var (
	newProposal           = market.ServiceProposal{ProviderID: "0x1"}
	newProposalPayload, _ = json.Marshal(newProposal)
)

func Test_NewRegistry(t *testing.T) {
	connection := nats.NewConnectionMock()

	assert.Equal(
		t,
		&registry{
			sender: nats.NewSender(connection, communication.NewCodecJSON(), "*"),
		},
		NewRegistry(connection),
	)
}

func Test_Registry_RegisterProposal(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	registry := NewRegistry(connection)
	err := registry.RegisterProposal(newProposal, &identity.SignerFake{})
	assert.NoError(t, err)

	assert.Equal(t, "*.proposal-register", connection.GetLastMessageSubject())
	assert.JSONEq(
		t,
		`{
			"proposal": `+string(newProposalPayload)+`
		}`,
		string(connection.GetLastMessage()),
	)
}

func Test_Registry_UnregisterProposal(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	registry := NewRegistry(connection)
	err := registry.UnregisterProposal(newProposal, &identity.SignerFake{})
	assert.NoError(t, err)

	assert.Equal(t, "*.proposal-unregister", connection.GetLastMessageSubject())
	assert.JSONEq(
		t,
		`{
			"proposal": `+string(newProposalPayload)+`
		}`,
		string(connection.GetLastMessage()),
	)
}

func Test_Registry_PingProposal(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	registry := NewRegistry(connection)
	err := registry.PingProposal(newProposal, &identity.SignerFake{})
	assert.NoError(t, err)

	assert.Equal(t, "*.proposal-ping", connection.GetLastMessageSubject())
	assert.JSONEq(
		t,
		`{
			"proposal": `+string(newProposalPayload)+`
		}`,
		string(connection.GetLastMessage()),
	)
}
