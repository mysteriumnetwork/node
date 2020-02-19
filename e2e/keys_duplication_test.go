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

package e2e

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/communication/nats/dialog"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/session"
	"github.com/stretchr/testify/assert"
)

func TestOpenVPNSessionKeyUniqueForSingleSession(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "mystE2E")
	defer os.RemoveAll(dir)

	assert.NoError(t, err)

	ks := initKeyStore(dir)

	session1, err1 := openvpnSessionConfig(ks)
	session2, err2 := openvpnSessionConfig(ks)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	assert.NotEqual(t, session1.ID, session2.ID)

	var cfg1, cfg2 openvpn.VPNConfig

	err1 = json.Unmarshal(session1.Config, &cfg1)
	err2 = json.Unmarshal(session2.Config, &cfg2)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	assert.NotEqual(t, cfg1.TLSPresharedKey, cfg2.TLSPresharedKey)
}

func openvpnSessionConfig(ks identity.Keystore) (s session.SessionDto, err error) {
	acc, err := ks.NewAccount("")
	if err != nil {
		return s, err
	}

	if err = ks.Unlock(acc, ""); err != nil {
		return s, err
	}

	signer := identity.NewSigner(ks, identity.FromAddress(acc.Address.String()))

	contact := market.Contact{
		Type: "nats/v1",
		Definition: nats_discovery.ContactNATSV1{
			Topic:           providerID + ".openvpn",
			BrokerAddresses: []string{"nats://broker:4222"},
		},
	}

	dialogEstablisher := dialog.NewDialogEstablisher(identity.FromAddress(acc.Address.String()), signer, nats.NewBrokerConnector())
	dialog, err := dialogEstablisher.EstablishDialog(identity.FromAddress(providerID), contact)
	if err != nil {
		return s, err
	}

	s, _, err = session.RequestSessionCreate(dialog, 1, nil, session.ConsumerInfo{})
	return s, err
}
