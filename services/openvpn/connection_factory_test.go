/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package openvpn

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server"
)

type cacheFake struct {
	location location.Location
	err      error
}

func (cf *cacheFake) Get() location.Location {
	return cf.location
}
func (cf *cacheFake) RefreshAndGet() (location.Location, error) {
	return cf.location, cf.err
}

func TestConnectionFactory_ImplementsConnectionConnectionCreatorInterface(t *testing.T) {
	var _ connection.ConnectionCreator = (*OpenvpnProcessBasedConnectionFactory)(nil)
}

func fakeSignerFactory(id identity.Identity) identity.Signer {
	return &identity.SignerFake{}
}

func TestConnectionFactory_ErrorsOnInvalidJson(t *testing.T) {
	clientFake := server.NewClientFake()
	factory := NewOpenvpnProcessBasedConnectionFactory(clientFake, "./", "./", "./", &fakeSessionStatsKeeper{}, &cacheFake{}, fakeSignerFactory)
	channel := make(chan connection.State)
	connectionOptions := connection.ConnectOptions{}
	_, err := factory.CreateConnection(connectionOptions, channel)
	assert.NotNil(t, err)
	assert.Equal(t, "unexpected end of JSON input", err.Error())
}

func TestConnectionFactory_CreatesConnection(t *testing.T) {
	clientFake := server.NewClientFake()
	factory := NewOpenvpnProcessBasedConnectionFactory(clientFake, "./", "./", "./", &fakeSessionStatsKeeper{}, &cacheFake{}, fakeSignerFactory)
	channel := make(chan connection.State)
	cfg := &VPNConfig{
		"1.2.3.4",
		10999,
		"tcp",
		tlsTestKey,
		caCertificate,
	}
	res, err := json.Marshal(cfg)
	assert.Nil(t, err)
	connectionOptions := connection.ConnectOptions{
		Config:     res,
		SessionID:  "some id",
		ConsumerID: identity.Identity{Address: "consumer"},
		ProviderID: identity.Identity{Address: "provider"},
	}
	conn, err := factory.CreateConnection(connectionOptions, channel)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
}
