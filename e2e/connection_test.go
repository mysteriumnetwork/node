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

package e2e

import (
	"flag"
	"github.com/cihub/seelog"
	"github.com/mysterium/node/tequilapi/client"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var host = flag.String("tequila.host", "localhost", "Specify tequila host for e2e tests")
var port = flag.Int("tequila.port", 4050, "Specify tequila port for e2e tests")

func TestClientConnectsToNode(t *testing.T) {
	//we cannot move tequilApi as var outside - host and port are not initialized yet :(
	//something related to how "go test" calls flag.Parse()
	tequilApi := client.NewClient(*host, *port)

	status, err := tequilApi.Status()
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", status.Status)

	identities, err := tequilApi.GetIdentities()
	assert.NoError(t, err)

	var identity client.IdentityDTO
	if len(identities) < 1 {
		identity, err = tequilApi.NewIdentity("")
		assert.NoError(t, err)
	} else {
		identity = identities[0]
	}
	seelog.Info("Client identity is: ", identity.Address)

	err = tequilApi.Unlock(identity.Address, "")
	assert.NoError(t, err)

	nonVpnIp, err := tequilApi.GetIP()
	assert.NoError(t, err)
	seelog.Info("Direct client address is: ", nonVpnIp)

	proposals, err := tequilApi.Proposals()
	if err != nil {
		assert.Error(t, err)
		assert.FailNow(t, "Proposals returned error - no point to continue")
	}

	//expect exactly one proposal
	if len(proposals) != 1 {
		assert.FailNow(t, "Exactly one proposal is expected - something is not right!")
	}

	proposal := proposals[0]
	seelog.Info("Selected proposal is: ", proposal)

	_, err = tequilApi.Connect(identity.Address, proposal.ProviderID)
	assert.NoError(t, err)

	time.Sleep(10 * time.Second)
	status, err = tequilApi.Status()
	assert.NoError(t, err)
	assert.Equal(t, "Connected", status.Status)

	vpnIp, err := tequilApi.GetIP()
	assert.NoError(t, err)
	seelog.Info("VPN client address is: ", vpnIp)

	err = tequilApi.Disconnect()
	assert.NoError(t, err)

	time.Sleep(10 * time.Second)
	status, err = tequilApi.Status()
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", status.Status)

}
