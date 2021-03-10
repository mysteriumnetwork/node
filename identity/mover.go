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
	"github.com/mysteriumnetwork/node/eventbus"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/pkg/errors"
)

// Mover can be used to move identities from and to keystore
// by exporting or importing them.
type Mover struct {
	handler       moverIdentityHandler
	ks            moverKeystore
	eventBus      eventbus.EventBus
	signerFactory SignerFactory
}

type moverKeystore interface {
	Delete(a accounts.Account, passphrase string) error
	Unlock(a accounts.Account, passphrase string) error
	Find(a accounts.Account) (accounts.Account, error)
	Export(a accounts.Account, passphrase, newPassphrase string) ([]byte, error)
	Import(keyJSON []byte, passphrase, newPassphrase string) (accounts.Account, error)
}

type moverIdentityHandler interface {
	IdentityExists(Identity, Signer) (bool, error)
}

// NewMover returns a new mover object.
func NewMover(ks moverKeystore, handler moverIdentityHandler, events eventbus.EventBus, signer SignerFactory) *Mover {
	return &Mover{
		ks:            ks,
		handler:       handler,
		eventBus:      events,
		signerFactory: signer,
	}
}

// Import imports a given blob as a new identity. It will return an
// error if that identity was never registered.
func (m *Mover) Import(blob []byte, currPass, newPass string) (Identity, error) {
	acc, err := m.ks.Import(blob, currPass, newPass)
	if err != nil {
		return Identity{}, err
	}

	if err := m.ks.Unlock(acc, newPass); err != nil {
		return Identity{}, err
	}

	identity := accountToIdentity(acc)
	if err := m.canImport(identity); err != nil {
		m.ks.Delete(acc, newPass)
		return Identity{}, err
	}

	m.eventBus.Publish(AppTopicIdentityCreated, identity.Address)
	return identity, nil
}

// Export exports a given identity and returns it as json blob.
func (m *Mover) Export(address, currPass, newPass string) ([]byte, error) {
	acc := addressToAccount(address)
	_, err := m.ks.Find(acc)
	if err != nil {
		return nil, errors.New("identity not found")
	}

	return m.ks.Export(acc, currPass, newPass)
}

func (m *Mover) canImport(id Identity) error {
	exists, err := m.handler.IdentityExists(id, m.signerFactory(id))
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("identity was never registered, can not import it")
	}

	return nil
}
