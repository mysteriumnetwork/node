/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package pingpong

import (
	"fmt"
	"sync"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
)

// ErrNotFound represents an error that indicates that there's no such invoice.
var ErrNotFound = errors.New("entry does not exist")

type bucketName string

const receivedInvoices bucketName = "received_invoices"
const sentInvoices bucketName = "sent_invoices"
const agreementRBucket bucketName = "agreement_r"

type genericInvoiceStorage interface {
	StoreInvoice(bucket string, key string, invoice crypto.Invoice) error
	GetInvoice(bucket string, key string) (crypto.Invoice, error)
}

type providerSpecificInvoiceStorage interface {
	genericInvoiceStorage
	StoreR(providerID identity.Identity, agreementID uint64, r string) error
	GetR(providerID identity.Identity, agreementID uint64) (string, error)
}

// ProviderInvoiceStorage allows the provider to store sent invoices.
type ProviderInvoiceStorage struct {
	gis providerSpecificInvoiceStorage
}

// NewProviderInvoiceStorage returns a new instance of provider invoice storage.
func NewProviderInvoiceStorage(gis providerSpecificInvoiceStorage) *ProviderInvoiceStorage {
	return &ProviderInvoiceStorage{
		gis: gis,
	}
}

// Store stores the given invoice.
func (pis *ProviderInvoiceStorage) Store(providerIdentity, consumerIdentity identity.Identity, invoice crypto.Invoice) error {
	return pis.gis.StoreInvoice(string(sentInvoices), providerIdentity.Address+consumerIdentity.Address, invoice)
}

// Get returns the stored invoice.
func (pis *ProviderInvoiceStorage) Get(providerIdentity, consumerIdentity identity.Identity) (crypto.Invoice, error) {
	return pis.gis.GetInvoice(string(sentInvoices), providerIdentity.Address+consumerIdentity.Address)
}

// StoreR stores the given R.
func (pis *ProviderInvoiceStorage) StoreR(providerID identity.Identity, agreementID uint64, r string) error {
	return pis.gis.StoreR(providerID, agreementID, r)
}

// GetR gets the R for agreement.
func (pis *ProviderInvoiceStorage) GetR(providerID identity.Identity, agreementID uint64) (string, error) {
	return pis.gis.GetR(providerID, agreementID)
}

type persistentStorage interface {
	GetValue(bucket string, key interface{}, to interface{}) error
	SetValue(bucket string, key interface{}, to interface{}) error
}

// InvoiceStorage allows to store promises.
type InvoiceStorage struct {
	bolt persistentStorage
	lock sync.Mutex
}

var errBoltNotFound = "not found"

// NewInvoiceStorage creates a new instance of invoice storage.
func NewInvoiceStorage(bolt persistentStorage) *InvoiceStorage {
	return &InvoiceStorage{
		bolt: bolt,
	}
}

// StoreInvoice stores the given invoice in the given bucket with the identity as key.
func (is *InvoiceStorage) StoreInvoice(bucket string, key string, invoice crypto.Invoice) error {
	is.lock.Lock()
	defer is.lock.Unlock()
	return errors.Wrap(is.bolt.SetValue(bucket, key, invoice), "could not save invoice")
}

func (is *InvoiceStorage) getRKey(providerID identity.Identity, agreementID uint64) string {
	return fmt.Sprintf("%v_%v", providerID.Address, agreementID)
}

// StoreR stores the given R.
func (is *InvoiceStorage) StoreR(providerID identity.Identity, agreementID uint64, r string) error {
	is.lock.Lock()
	defer is.lock.Unlock()
	err := is.bolt.SetValue(string(agreementRBucket), is.getRKey(providerID, agreementID), r)
	return errors.Wrap(err, "could not save R")
}

// GetR returns the saved R.
func (is *InvoiceStorage) GetR(providerID identity.Identity, agreementID uint64) (string, error) {
	is.lock.Lock()
	defer is.lock.Unlock()
	var r string
	err := is.bolt.GetValue(string(agreementRBucket), is.getRKey(providerID, agreementID), &r)
	if err != nil {
		// wrap the error to an error we can check for
		if err.Error() == errBoltNotFound {
			return "", ErrNotFound
		}
		return r, errors.Wrap(err, "could not get r")
	}
	return r, nil
}

// GetInvoice gets the corresponding invoice from storage.
func (is *InvoiceStorage) GetInvoice(bucket string, key string) (crypto.Invoice, error) {
	is.lock.Lock()
	defer is.lock.Unlock()
	invoice := &crypto.Invoice{}
	err := is.bolt.GetValue(bucket, key, invoice)
	if err != nil {
		// wrap the error to an error we can check for
		if err.Error() == errBoltNotFound {
			err = ErrNotFound
		} else {
			err = errors.Wrap(err, "could not get invoice")
		}
	}
	return *invoice, err
}
