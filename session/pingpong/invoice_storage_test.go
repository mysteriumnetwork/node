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
	"math/big"
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
	AgreementID:    big.NewInt(1),
	AgreementTotal: big.NewInt(1),
	TransactorFee:  big.NewInt(1),
	Hashlock:       "hashlock1",
	Provider:       identityOne.Address,
}

var invoiceTwo = crypto.Invoice{
	AgreementID:    big.NewInt(2),
	AgreementTotal: big.NewInt(2),
	TransactorFee:  big.NewInt(2),
	Hashlock:       "hashlock2",
	Provider:       identityTwo.Address,
}

func TestProviderInvoiceStorage(t *testing.T) {
	providerID := identity.FromAddress("0xprovider")
	dir, err := os.MkdirTemp("", "providerInvoiceTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	defer bolt.Close()

	genericStorage := NewInvoiceStorage(bolt)

	providerStorage := NewProviderInvoiceStorage(genericStorage)

	// check if errors are wrapped correctly
	_, err = providerStorage.Get(providerID, identityOne)
	assert.Equal(t, ErrNotFound, err)

	// store and check that invoice is stored correctly
	err = providerStorage.Store(providerID, identityOne, invoiceOne)
	assert.NoError(t, err)

	invoice, err := providerStorage.Get(providerID, identityOne)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceOne, invoice)

	// overwrite the invoice, check if it is overwritten
	err = providerStorage.Store(providerID, identityOne, invoiceTwo)
	assert.NoError(t, err)

	invoice, err = providerStorage.Get(providerID, identityOne)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceTwo, invoice)

	// store two invoices, check if both are gotten correctly
	err = providerStorage.Store(providerID, identityTwo, invoiceOne)
	assert.NoError(t, err)

	invoice, err = providerStorage.Get(providerID, identityOne)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceTwo, invoice)

	invoice, err = providerStorage.Get(providerID, identityTwo)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceOne, invoice)

	// check if multiple providers can actually store their invoices
	providerTwo := identity.FromAddress("0xproviderTwo")

	// store and check that invoice is stored correctly
	err = providerStorage.Store(providerTwo, identityOne, invoiceOne)
	assert.NoError(t, err)

	invoice, err = providerStorage.Get(providerTwo, identityOne)
	assert.NoError(t, err)
	assert.EqualValues(t, invoiceOne, invoice)

	_, err = providerStorage.Get(providerTwo, identityTwo)
	assert.Equal(t, ErrNotFound, err)

	// test R storage
	var agreementID1 = big.NewInt(1)
	r1 := "my r"
	err = providerStorage.StoreR(providerID, agreementID1, r1)
	assert.NoError(t, err)

	var agreementID2 = big.NewInt(1222)
	r2 := "my other r"
	err = providerStorage.StoreR(providerID, agreementID2, r2)
	assert.NoError(t, err)

	r, err := providerStorage.GetR(providerID, agreementID2)
	assert.NoError(t, err)
	assert.Equal(t, r2, r)

	r, err = providerStorage.GetR(providerID, agreementID1)
	assert.NoError(t, err)
	assert.Equal(t, r1, r)

	// check if multiple providers can actually store their R's
	r, err = providerStorage.GetR(providerTwo, agreementID2)
	assert.Equal(t, ErrNotFound, err)

	err = providerStorage.StoreR(providerTwo, agreementID2, r2)
	assert.NoError(t, err)

	r, err = providerStorage.GetR(providerID, agreementID2)
	assert.NoError(t, err)
	assert.Equal(t, r2, r)
}
