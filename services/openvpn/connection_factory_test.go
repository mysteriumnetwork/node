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

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

var fakeSessionConfig = []byte(`{
    "remote": "1.2.3.4",
    "port": 10999,
    "protocol": "tcp",
    "TLSPresharedKey": "\n-----BEGIN OpenVPN Static key V1-----\n7573bf79ebecb38d2a009d28830ecf5b0b11e27362513fe4b09b55f07054c4c7c3cebeb00bf8bb2d05cfa0f79308e762e684b931db2179e7a21618ea\n869cbb5b1b9753ca05d3b87708389ccc154c9278a92964002ea888c1011fb06444088162ff6a4c1d5a8ee0ab30fd1b4dc9aaaa8c8901b426d25063cc\n660d47103ff14e2cae99ca9ce28d70f927d090c144c49b3d86832c1e1c67562a6d248dff8a2583948a065015ec84d8d7bfe63385e257a6338471e2c6\n7075416f4771beb0c872cc09c9ce4318fd8c9446987664f04ceeeb4e3c49f7101aa4953795014696a2f4e1cb129127fe5830627563efb127589b3693\naddc15c1393f4db6c7f8d55ba598fbe5\n-----END OpenVPN Static key V1-----\n",
    "CACertificate": "\n-----BEGIN CERTIFICATE-----\nMIIByDCCAW6gAwIBAgICBFcwCgYIKoZIzj0EAwIwQzELMAkGA1UEBhMCR0IxGzAZ\nBgNVBAoTEk15c3Rlcm1pdW0ubmV0d29yazEXMBUGA1UECxMOTXlzdGVyaXVtIFRl\nYW0wHhcNMTgwNTA4MTIwMDU5WhcNMjgwNTA4MTIwMDU5WjBDMQswCQYDVQQGEwJH\nQjEbMBkGA1UEChMSTXlzdGVybWl1bS5uZXR3b3JrMRcwFQYDVQQLEw5NeXN0ZXJp\ndW0gVGVhbTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABKvoBgL5PCWlUr4PSl2j\njSXtW8ohVESWVL6l0de+Sj6dWsjELxmLAKdnwep9CcYvGE0i3Q0M24C/ZSoCREpl\n8UOjUjBQMA4GA1UdDwEB/wQEAwIChDAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYB\nBQUHAwEwDwYDVR0TAQH/BAUwAwEB/zAOBgNVHQ4EBwQFAQIDBAUwCgYIKoZIzj0E\nAwIDSAAwRQIhAKLOIPprhU7CCyFG52J8FmyzwBJjcwHu+ZzGFrdfwEKKAiB7xkYM\nYFcPCscvdnZ1U8hTUaREZmDB2w9eaGyCM4YXAg==\n-----END CERTIFICATE-----\n"
}`)

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

var _ connection.Factory = &ProcessBasedConnectionFactory{}

func fakeSignerFactory(_ identity.Identity) identity.Signer {
	return &identity.SignerFake{}
}

func TestConnectionFactory_ErrorsOnInvalidConfig(t *testing.T) {
	factory := NewProcessBasedConnectionFactory("./", "./", "./", &cacheFake{}, fakeSignerFactory)
	channel := make(chan connection.State)
	statisticsChannel := make(chan consumer.SessionStatistics)
	connectionOptions := connection.ConnectOptions{}
	conn, err := factory.Create(channel, statisticsChannel)
	assert.Nil(t, err)
	err = conn.Start(connectionOptions)
	assert.EqualError(t, err, "unexpected end of JSON input")
}

func TestConnectionFactory_CreatesConnection(t *testing.T) {
	factory := NewProcessBasedConnectionFactory("./", "./", "./", &cacheFake{}, fakeSignerFactory)
	channel := make(chan connection.State)
	statisticsChannel := make(chan consumer.SessionStatistics)
	conn, err := factory.Create(channel, statisticsChannel)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
}
