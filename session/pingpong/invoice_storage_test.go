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
	"io/ioutil"
	"os"
	"testing"

	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

var identityOne = identity.FromAddress("0xf3d7B9597F8c4137Fa9F73e1Db533CEDC61e844f")
var identityTwo = identity.FromAddress("0x3D32e7D08BE7E5c3870679b3A7Ef60e9422196Ab")

var invoiceOne = crypto.Invoice{
	AgreementID:    1,
	AgreementTotal: 1,
	Fee:            1,
	Hashlock:       "hashlock1",
	Provider:       identityOne.Address,
}

var invoiceTwo = crypto.Invoice{
	AgreementID:    2,
	AgreementTotal: 2,
	Fee:            2,
	Hashlock:       "hashlock2",
	Provider:       identityTwo.Address,
}

func TestConsumerInvoiceStorage(t *testing.T) {
	dir, err := ioutil.TempDir("", "consumerInvoiceTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	defer bolt.Close()

	genericStorage := NewInvoiceStorage(bolt)

	consumerStorage := NewConsumerInvoiceStorage(genericStorage)

	// check if errors are wrapped correctly
	_, err = consumerStorage.Get(identityOne)
	assert.Equal(t, ErrNotFound, err)

	// store and check that invoice is stored correctly
	err = consumerStorage.Store(identityOne, invoiceOne)
	assert.NoError(t, err)

	invoice, err := consumerStorage.Get(identityOne)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceOne, invoice)

	// overwrite the invoice, check if it is overwritten
	err = consumerStorage.Store(identityOne, invoiceTwo)
	assert.NoError(t, err)

	invoice, err = consumerStorage.Get(identityOne)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceTwo, invoice)

	// store two invoices, check if both are gotten correctly
	err = consumerStorage.Store(identityTwo, invoiceOne)
	assert.NoError(t, err)

	invoice, err = consumerStorage.Get(identityOne)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceTwo, invoice)

	invoice, err = consumerStorage.Get(identityTwo)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceOne, invoice)
}

func TestProviderInvoiceStorage(t *testing.T) {
	dir, err := ioutil.TempDir("", "providerInvoiceTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	defer bolt.Close()

	genericStorage := NewInvoiceStorage(bolt)

	providerStorage := NewProviderInvoiceStorage(genericStorage)

	// check if errors are wrapped correctly
	_, err = providerStorage.Get(identityOne)
	assert.Equal(t, ErrNotFound, err)

	// store and check that invoice is stored correctly
	err = providerStorage.Store(identityOne, invoiceOne)
	assert.NoError(t, err)

	invoice, err := providerStorage.Get(identityOne)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceOne, invoice)

	// overwrite the invoice, check if it is overwritten
	err = providerStorage.Store(identityOne, invoiceTwo)
	assert.NoError(t, err)

	invoice, err = providerStorage.Get(identityOne)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceTwo, invoice)

	// store two invoices, check if both are gotten correctly
	err = providerStorage.Store(identityTwo, invoiceOne)
	assert.NoError(t, err)

	invoice, err = providerStorage.Get(identityOne)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceTwo, invoice)

	invoice, err = providerStorage.Get(identityTwo)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceOne, invoice)
}
