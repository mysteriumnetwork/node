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
const agreementIDCounter bucketName = "agreement_id_counter"
const agreementRBucket bucketName = "agreement_r"

type genericInvoiceStorage interface {
	StoreInvoice(bucket string, identity identity.Identity, invoice crypto.Invoice) error
	GetInvoice(bucket string, identity identity.Identity) (crypto.Invoice, error)
}

type providerSpecificInvoiceStorage interface {
	genericInvoiceStorage
	GetNewAgreementID() (uint64, error)
	StoreR(agreementID uint64, r string) error
	GetR(agreementID uint64) (string, error)
}

// ConsumerInvoiceStorage allows the consumer to store received invoices.
type ConsumerInvoiceStorage struct {
	gis genericInvoiceStorage
}

// NewConsumerInvoiceStorage allows the consumer to store invoices.
func NewConsumerInvoiceStorage(gis genericInvoiceStorage) *ConsumerInvoiceStorage {
	return &ConsumerInvoiceStorage{
		gis: gis,
	}
}

// Store stores the given invoice
func (cis *ConsumerInvoiceStorage) Store(providerIdentity identity.Identity, invoice crypto.Invoice) error {
	return cis.gis.StoreInvoice(string(receivedInvoices), providerIdentity, invoice)
}

// Get returns the stored invoice
func (cis *ConsumerInvoiceStorage) Get(providerIdentity identity.Identity) (crypto.Invoice, error) {
	return cis.gis.GetInvoice(string(receivedInvoices), providerIdentity)
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
func (pis *ProviderInvoiceStorage) Store(consumerIdentity identity.Identity, invoice crypto.Invoice) error {
	return pis.gis.StoreInvoice(string(sentInvoices), consumerIdentity, invoice)
}

// Get returns the stored invoice.
func (pis *ProviderInvoiceStorage) Get(consumerIdentity identity.Identity) (crypto.Invoice, error) {
	return pis.gis.GetInvoice(string(sentInvoices), consumerIdentity)
}

// GetNewAgreementID returns a new agreement id for the provider.
func (pis *ProviderInvoiceStorage) GetNewAgreementID() (uint64, error) {
	return pis.gis.GetNewAgreementID()
}

// StoreR stores the given R.
func (pis *ProviderInvoiceStorage) StoreR(agreementID uint64, r string) error {
	return pis.gis.StoreR(agreementID, r)
}

// GetR gets the R for agreement.
func (pis *ProviderInvoiceStorage) GetR(agreementID uint64) (string, error) {
	return pis.gis.GetR(agreementID)
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
func (is *InvoiceStorage) StoreInvoice(bucket string, identity identity.Identity, invoice crypto.Invoice) error {
	is.lock.Lock()
	defer is.lock.Unlock()
	return errors.Wrap(is.bolt.SetValue(bucket, identity.Address, invoice), "could not save invoice")
}

// GetNewAgreementID generates a new agreement id.
func (is *InvoiceStorage) GetNewAgreementID() (uint64, error) {
	is.lock.Lock()
	defer is.lock.Unlock()

	var res uint64 = 1
	err := is.bolt.GetValue(string(agreementIDCounter), "provider_agreement_id", &res)
	if err != nil {
		if err.Error() != errBoltNotFound {
			return res, errors.Wrap(err, "could not get agreement id")
		}
	}
	err = is.bolt.SetValue(string(agreementIDCounter), "provider_agreement_id", res+1)
	if err != nil {
		err = errors.Wrap(err, "could not set agreement id")
	}
	return res, err
}

// StoreR stores the given R.
func (is *InvoiceStorage) StoreR(agreementID uint64, r string) error {
	is.lock.Lock()
	defer is.lock.Unlock()
	err := is.bolt.SetValue(string(agreementRBucket), agreementID, r)
	return errors.Wrap(err, "could not save R")
}

// GetR returns the saved R.
func (is *InvoiceStorage) GetR(agreementID uint64) (string, error) {
	is.lock.Lock()
	defer is.lock.Unlock()
	var r string
	err := is.bolt.GetValue(string(agreementRBucket), agreementID, &r)
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
func (is *InvoiceStorage) GetInvoice(bucket string, identity identity.Identity) (crypto.Invoice, error) {
	is.lock.Lock()
	defer is.lock.Unlock()
	invoice := &crypto.Invoice{}
	err := is.bolt.GetValue(bucket, identity.Address, invoice)
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
