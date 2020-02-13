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
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

func fakeSignerFactory(_ identity.Identity) identity.Signer {
	return &identity.SignerFake{}
}

func TestConnection_ErrorsOnInvalidConfig(t *testing.T) {
	conn, err := NewClient("./", "./", "./", fakeSignerFactory, ip.NewResolverMock("1.1.1.1"), &MockNATPinger{}, time.Second)
	connectionOptions := connection.ConnectOptions{}
	assert.Nil(t, err)
	err = conn.Start(connectionOptions)
	assert.EqualError(t, err, "unexpected end of JSON input")
}

func TestConnection_CreatesConnection(t *testing.T) {
	conn, err := NewClient("./", "./", "./", fakeSignerFactory, ip.NewResolverMock("1.1.1.1"), &MockNATPinger{}, time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
}

// MockNATPinger returns a mock nat pinger, that really doesnt do much
type MockNATPinger struct{}

// PingProvider does nothing
func (mnp *MockNATPinger) PingProvider(_ string, port, consumerPort, proxyPort int, _ <-chan struct{}) error {
	return nil
}
