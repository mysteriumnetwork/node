/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package identity

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	ethKs "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/stretchr/testify/assert"
)

var (
	idAddress = "0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"
	idAccount = accounts.Account{
		Address: common.HexToAddress("53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"),
	}
	idChainID int64 = 1
	idKey, _        = crypto.HexToECDSA("6f88637b68ee88816e73f663aef709d7009836c98ae91ef31e3dfac7be3a1657")
)

func Test_UnlockAndSignAndVerify(t *testing.T) {
	ks := NewKeystoreFilesystem("dir", &ethKeystoreMock{account: idAccount})
	ks.loadKey = func(addr common.Address, filename, auth string) (*ethKs.Key, error) {
		return &ethKs.Key{Address: addr, PrivateKey: idKey}, nil
	}

	bus := eventbus.New()
	manager := NewIdentityManager(ks, bus, NewResidentCountry(bus, newMockLocationResolver("LT")))
	err := manager.Unlock(idChainID, idAddress, "")
	assert.NoError(t, err)

	signer := NewSigner(ks, FromAddress(idAddress))
	signature, err := signer.Sign([]byte("Boop!"))
	assert.NoError(t, err)
	assert.Exactly(
		t,
		SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000"),
		signature,
	)

	verifier := NewVerifierIdentity(FromAddress(idAddress))
	res, _ := verifier.Verify([]byte("Boop!"), signature)
	assert.True(t, res)
}
