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
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

type MockPeerInvoiceSender struct {
	mockError     error
	chanToWriteTo chan crypto.Invoice
}

func (mpis *MockPeerInvoiceSender) Send(invoice crypto.Invoice) error {
	if mpis.chanToWriteTo != nil {
		mpis.chanToWriteTo <- invoice
	}
	return mpis.mockError
}

type mockAccountantCaller struct{}

func (mac *mockAccountantCaller) RequestPromise(em crypto.ExchangeMessage) (crypto.Promise, error) {
	return crypto.Promise{}, nil
}

func (mac *mockAccountantCaller) RevealR(r string, provider string, agreementID uint64) error {
	return nil
}

func Test_InvoiceTracker_Start_Stop(t *testing.T) {
	dir, err := ioutil.TempDir("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	mockSender := &MockPeerInvoiceSender{
		chanToWriteTo: make(chan crypto.Invoice, 10),
	}

	exchangeMessageChan := make(chan crypto.ExchangeMessage)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(time.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	accountantPromiseStorage := NewAccountantPromiseStorage(bolt)
	deps := InvoiceTrackerDeps{
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		PaymentInfo:                dto.PaymentPerTime{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		AccountantID:               identity.FromAddress(acc.Address.Hex()),
		AccountantCaller:           &mockAccountantCaller{},
		AccountantPromiseStorage:   accountantPromiseStorage,
	}
	invoiceTracker := NewInvoiceTracker(deps)

	go func() {
		time.Sleep(time.Nanosecond * 10)
		invoiceTracker.Stop()
	}()

	err = invoiceTracker.Start()
	assert.Nil(t, err)
}

func Test_InvoiceTracker_BubblesErrors(t *testing.T) {
	dir, err := ioutil.TempDir("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	mockSender := &MockPeerInvoiceSender{
		chanToWriteTo: make(chan crypto.Invoice, 10),
	}

	exchangeMessageChan := make(chan crypto.ExchangeMessage)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(time.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	accountantPromiseStorage := NewAccountantPromiseStorage(bolt)
	deps := InvoiceTrackerDeps{
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		PaymentInfo:                dto.PaymentPerTime{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		AccountantID:               identity.FromAddress(acc.Address.Hex()),
		AccountantCaller:           &mockAccountantCaller{},
		AccountantPromiseStorage:   accountantPromiseStorage,
	}
	invoiceTracker := NewInvoiceTracker(deps)

	defer invoiceTracker.Stop()

	errChan := make(chan error)
	go func() { errChan <- invoiceTracker.Start() }()

	exchangeMessageChan <- crypto.ExchangeMessage{}

	err = <-errChan
	assert.Error(t, err)
}

func Test_InvoiceTracker_SendsInvoice(t *testing.T) {
	dir, err := ioutil.TempDir("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	mockSender := &MockPeerInvoiceSender{
		chanToWriteTo: make(chan crypto.Invoice, 10),
	}

	exchangeMessageChan := make(chan crypto.ExchangeMessage)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(time.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	accountantPromiseStorage := NewAccountantPromiseStorage(bolt)
	deps := InvoiceTrackerDeps{
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		PaymentInfo:                dto.PaymentPerTime{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		AccountantID:               identity.FromAddress(acc.Address.Hex()),
		AccountantCaller:           &mockAccountantCaller{},
		AccountantPromiseStorage:   accountantPromiseStorage,
	}
	invoiceTracker := NewInvoiceTracker(deps)
	defer invoiceTracker.Stop()

	errChan := make(chan error)
	go func() { errChan <- invoiceTracker.Start() }()

	invoice := <-mockSender.chanToWriteTo
	assert.Equal(t, uint64(1), invoice.AgreementID)
	assert.True(t, invoice.AgreementTotal > 0)
	assert.Len(t, invoice.Hashlock, 64)
	assert.Equal(t, strings.ToLower(acc.Address.Hex()), strings.ToLower(invoice.Provider))
}

func Test_calculateMaxNotReceivedExchangeMessageCount(t *testing.T) {
	res := calculateMaxNotReceivedExchangeMessageCount(time.Minute*5, time.Second*240)
	assert.Equal(t, uint64(1), res)
	res = calculateMaxNotReceivedExchangeMessageCount(time.Minute*5, time.Second*20)
	assert.Equal(t, uint64(15), res)
	res = calculateMaxNotReceivedExchangeMessageCount(time.Hour*2, time.Second*20)
	assert.Equal(t, uint64(360), res)
}
