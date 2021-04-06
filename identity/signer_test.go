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
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/eventbus"
)

var (
	signerAddress = "53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"
	signerAccount = accounts.Account{
		Address: common.HexToAddress("53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"),
	}
	signerChainID int64 = 1
	signerKey, _        = crypto.HexToECDSA("6f88637b68ee88816e73f663aef709d7009836c98ae91ef31e3dfac7be3a1657")
)

func TestSigningMessageWithUnlockedAccount(t *testing.T) {
	ks := NewKeystoreFilesystem("dir", &ethKeystoreMock{account: signerAccount})
	ks.loadKey = func(addr common.Address, filename, auth string) (*ethKs.Key, error) {
		return &ethKs.Key{Address: addr, PrivateKey: signerKey}, nil
	}

	bus := eventbus.New()
	manager := NewIdentityManager(ks, bus, NewResidentCountry(bus, newMockLocationResolver("LT")))
	err := manager.Unlock(signerChainID, signerAddress, "")
	assert.NoError(t, err)

	signer := NewSigner(ks, FromAddress(signerAddress))
	message := []byte("MystVpnSessionId:Boop!")
	signature, err := signer.Sign([]byte(message))
	signatureBase64 := signature.Base64()
	t.Logf("signature in base64: %s", signatureBase64)
	assert.NoError(t, err)
	assert.Equal(
		t,
		SignatureBase64("V6ifmvLuAT+hbtLBX/0xm3C0afywxTIdw1HqLmA4onpwmibHbxVhl50Gr3aRUZMqw1WxkfSIVdhpbCluHGBKsgE="),
		signature,
	)
}
