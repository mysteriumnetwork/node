/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/stretchr/testify/assert"
)

type moverIdentityHandlerMock struct {
	exists bool
}

func (m *moverIdentityHandlerMock) IdentityExists(_ Identity, _ Signer) (bool, error) {
	return m.exists, nil
}

type fakeSigner struct {
}

func (fs *fakeSigner) Sign(message []byte) (Signature, error) {
	return SignatureBase64("deadbeef"), nil
}

var fakeSignerFactory = func(id Identity) Signer {
	return &fakeSigner{}
}

func TestMoverImport(t *testing.T) {
	ks := &ethKeystoreMock{account: encryptionAccount}
	bus := eventbus.New()
	m := NewMover(ks, bus, fakeSignerFactory)

	t.Run("identity import green path", func(t *testing.T) {
		got, err := m.Import([]byte(""), "asdf", "asdf")
		assert.NoError(t, err)
		assert.Equal(t, accountToIdentity(encryptionAccount).Address, got.Address)
		assert.True(t, ks.unlocked)
	})

	t.Run("identity was never registered should succeed", func(t *testing.T) {
		_, err := m.Import([]byte(""), "asdf", "asdf")
		assert.NoError(t, err)
	})

}

func TestMoverExport(t *testing.T) {
	ks := &ethKeystoreMock{account: encryptionAccount}
	bus := eventbus.New()
	m := NewMover(ks, bus, fakeSignerFactory)

	t.Run("identity export green path", func(t *testing.T) {
		blob, err := m.Export(accountToIdentity(encryptionAccount).Address, "qwe", "qwe")
		assert.NoError(t, err)
		assert.Equal(t, "exported", string(blob))
	})

	t.Run("identity export not found", func(t *testing.T) {
		blob, err := m.Export(accountToIdentity(accounts.Account{}).Address, "qwe", "qwe")
		assert.Nil(t, blob)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
