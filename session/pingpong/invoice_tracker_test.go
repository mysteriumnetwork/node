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

const mockRegistryAddress = "0xE6b3a5c92e7c1f9543A0aEE9A93fE2F6B584c1f7"
const mockChannelImplementation = "0xa26b684d8dBa935DD34544FBd3Ab4d7FDe1C4D07"

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

func generateExchangeMessage(t *testing.T, amount uint64, invoice crypto.Invoice, channel string) (crypto.ExchangeMessage, string) {
	dir, err := ioutil.TempDir("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	err = ks.Unlock(acc, "")
	assert.Nil(t, err)

	if channel == "" {
		addr, err := crypto.GenerateChannelAddress(acc.Address.Hex(), mockRegistryAddress, mockChannelImplementation)
		assert.Nil(t, err)
		channel = addr
	}

	em, err := crypto.CreateExchangeMessage(invoice, amount, channel, ks, acc.Address)
	assert.Nil(t, err)
	if em != nil {
		return *em, acc.Address.Hex()
	}
	return crypto.ExchangeMessage{}, acc.Address.Hex()
}

func TestInvoiceTracker_receiveExchangeMessageOrTimeout(t *testing.T) {
	msg1, addr1 := generateExchangeMessage(t, 10, crypto.Invoice{AgreementTotal: 10}, "some channel")
	msg2, addr2 := generateExchangeMessage(t, 10, crypto.Invoice{AgreementTotal: 10, Hashlock: "hashlock"}, "some channel")
	msg3, addr3 := generateExchangeMessage(t, 10, crypto.Invoice{AgreementTotal: 10, Hashlock: "hashlock"}, "")
	type fields struct {
		peer                       identity.Identity
		exchangeMessageChan        chan crypto.ExchangeMessage
		exchangeMessageWaitTimeout time.Duration
		accountantFailureCount     uint64
		accountantPromiseStorage   accountantPromiseStorage
		accountantID               identity.Identity
		lastInvoice                lastInvoice
		lastExchangeMessage        crypto.ExchangeMessage
		accountantCaller           accountantCaller
		channelImplementation      string
		registryAddress            string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		em      *crypto.ExchangeMessage
	}{
		{
			name:    "errors on timeout",
			wantErr: true,
		},
		{
			name:    "errors on invalid signature",
			wantErr: true,
			fields: fields{
				exchangeMessageWaitTimeout: time.Minute,
				exchangeMessageChan:        make(chan crypto.ExchangeMessage),
				peer:                       identity.FromAddress(addr1),
			},
			em: &crypto.ExchangeMessage{},
		},
		{
			name:    "errors on missmatching hashlocks",
			wantErr: true,
			fields: fields{
				exchangeMessageWaitTimeout: time.Minute,
				exchangeMessageChan:        make(chan crypto.ExchangeMessage),
				peer:                       identity.FromAddress(addr1),
			},
			em: &msg1,
		},
		{
			name:    "errors on bad channel ID",
			wantErr: true,
			fields: fields{
				exchangeMessageWaitTimeout: time.Minute,
				exchangeMessageChan:        make(chan crypto.ExchangeMessage),
				lastInvoice: lastInvoice{
					invoice: crypto.Invoice{AgreementTotal: 10, Hashlock: "hashlock"},
				},
				peer: identity.FromAddress(addr2),
			},
			em: &msg2,
		},
		{
			name:    "completes green path",
			wantErr: false,
			fields: fields{
				exchangeMessageWaitTimeout: time.Minute,
				exchangeMessageChan:        make(chan crypto.ExchangeMessage),
				lastInvoice: lastInvoice{
					invoice: crypto.Invoice{AgreementTotal: 10, Hashlock: "hashlock"},
				},
				accountantCaller:         &mockAccountantCaller{},
				accountantPromiseStorage: &mockAccountantPromiseStorage{},
				peer:                     identity.FromAddress(addr3),
				registryAddress:          mockRegistryAddress,
				channelImplementation:    mockChannelImplementation,
			},
			em: &msg3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := &InvoiceTracker{
				peer:                       tt.fields.peer,
				exchangeMessageChan:        tt.fields.exchangeMessageChan,
				exchangeMessageWaitTimeout: tt.fields.exchangeMessageWaitTimeout,
				accountantFailureCount:     tt.fields.accountantFailureCount,
				accountantPromiseStorage:   tt.fields.accountantPromiseStorage,
				accountantID:               tt.fields.accountantID,
				lastExchangeMessage:        tt.fields.lastExchangeMessage,
				accountantCaller:           tt.fields.accountantCaller,
				lastInvoice:                tt.fields.lastInvoice,
				registryAddress:            tt.fields.registryAddress,
				channelImplementation:      tt.fields.channelImplementation,
			}
			if tt.em != nil {
				go func() {
					time.Sleep(time.Millisecond * 5)
					tt.fields.exchangeMessageChan <- *tt.em
				}()
			}
			if err := it.receiveExchangeMessageOrTimeout(); (err != nil) != tt.wantErr {
				t.Errorf("InvoiceTracker.receiveExchangeMessageOrTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockAccountantPromiseStorage struct {
}

func (maps *mockAccountantPromiseStorage) Store(accountantID identity.Identity, promise crypto.Promise) error {
	return nil
}

func (maps *mockAccountantPromiseStorage) Get(accountantID identity.Identity) (crypto.Promise, error) {
	return crypto.Promise{}, nil
}
