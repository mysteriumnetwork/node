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
 * along with this prograi.  If not, see <http://www.gnu.org/licenses/>.
 */

package identity

import (
	"errors"

	"github.com/mysteriumnetwork/node/eventbus"

	"github.com/ethereum/go-ethereum/accounts"
)

// Mover is wrapper on both the Exporter and Importer
// and can be used to manipulate private keys in to either direction.
type Mover struct {
	*Exporter
	*Importer
}

type moverKeystore interface {
	Unlock(a accounts.Account, passphrase string) error
	Find(a accounts.Account) (accounts.Account, error)
	Export(a accounts.Account, passphrase, newPassphrase string) ([]byte, error)
	Import(keyJSON []byte, passphrase, newPassphrase string) (accounts.Account, error)
}

type moverIdentityHandler interface {
	IdentityExists(Identity, Signer) (bool, error)
}

// NewMover returns a new mover object.
func NewMover(ks moverKeystore, events eventbus.EventBus, signer SignerFactory) *Mover {
	return &Mover{
		Exporter: NewExporter(ks),
		Importer: NewImporter(ks, events, signer),
	}
}

// Importer exposes a way to import an private keys.
type Importer struct {
	ks            moverKeystore
	eventBus      eventbus.EventBus
	signerFactory SignerFactory
}

// NewImporter returns a new importer object.
func NewImporter(ks moverKeystore, events eventbus.EventBus, signer SignerFactory) *Importer {
	return &Importer{
		ks:            ks,
		eventBus:      events,
		signerFactory: signer,
	}
}

// Import imports a given blob as a new identity. It will return an
// error if that identity was never registered.
func (i *Importer) Import(blob []byte, currPass, newPass string) (Identity, error) {
	acc, err := i.ks.Import(blob, currPass, newPass)
	if err != nil {
		return Identity{}, err
	}

	if err := i.ks.Unlock(acc, newPass); err != nil {
		return Identity{}, err
	}

	identity := accountToIdentity(acc)
	i.eventBus.Publish(AppTopicIdentityCreated, identity.Address)
	return identity, nil
}

// Exporter exposes a way to export private keys.
type Exporter struct {
	ks moverKeystore
}

// NewExporter returns a new exporter object.
func NewExporter(ks moverKeystore) *Exporter {
	return &Exporter{
		ks: ks,
	}
}

// Export exports a given identity and returns it as json blob.
func (e *Exporter) Export(address, currPass, newPass string) ([]byte, error) {
	acc := addressToAccount(address)
	_, err := e.ks.Find(acc)
	if err != nil {
		return nil, errors.New("identity not found")
	}

	return e.ks.Export(acc, currPass, newPass)
}
