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
	"encoding/hex"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const mockRegistryAddress = "0xE6b3a5c92e7c1f9543A0aEE9A93fE2F6B584c1f7"
const mockAccountantAddress = "0xf28DB7aDf64A2811202B149aa4733A1FB9100e5c"
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
		Publisher:                  &mockPublisher{},
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		PaymentInfo:                dto.PaymentRate{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		AccountantID:               identity.FromAddress(acc.Address.Hex()),
		AccountantCaller:           &mockAccountantCaller{},
		FeeProvider:                &mockTransactor{},
		AccountantPromiseStorage:   accountantPromiseStorage,
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		BlockchainHelper:           &mockBlockchainHelper{isRegistered: true},
	}
	invoiceTracker := NewInvoiceTracker(deps)

	go func() {
		time.Sleep(time.Nanosecond * 10)
		invoiceTracker.Stop()
	}()

	err = invoiceTracker.Start()
	assert.Nil(t, err)
}

func Test_InvoiceTracker_Start_RefusesUnregisteredUser(t *testing.T) {
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
		Publisher:                  &mockPublisher{},
		ExchangeMessageWaitTimeout: time.Second,
		PaymentInfo:                dto.PaymentRate{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		AccountantID:               identity.FromAddress(acc.Address.Hex()),
		AccountantCaller:           &mockAccountantCaller{},
		AccountantPromiseStorage:   accountantPromiseStorage,
		BlockchainHelper:           &mockBlockchainHelper{isRegistered: false},
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		FeeProvider:                &mockTransactor{},
	}
	invoiceTracker := NewInvoiceTracker(deps)

	go func() {
		time.Sleep(time.Nanosecond * 10)
		invoiceTracker.Stop()
	}()

	err = invoiceTracker.Start()
	assert.Equal(t, ErrConsumerNotRegistered, err)
}

func Test_InvoiceTracker_Start_BubblesRegistrationCheckErrors(t *testing.T) {
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

	mockError := errors.New("explosions everywhere")
	tracker := session.NewTracker(time.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	accountantPromiseStorage := NewAccountantPromiseStorage(bolt)
	deps := InvoiceTrackerDeps{
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		Publisher:                  &mockPublisher{},
		ChargePeriod:               time.Nanosecond,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		PaymentInfo:                dto.PaymentRate{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		AccountantID:               identity.FromAddress(acc.Address.Hex()),
		AccountantCaller:           &mockAccountantCaller{},
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		AccountantPromiseStorage:   accountantPromiseStorage,
		BlockchainHelper:           &mockBlockchainHelper{isRegisteredError: mockError},
		FeeProvider:                &mockTransactor{},
	}
	invoiceTracker := NewInvoiceTracker(deps)

	go func() {
		time.Sleep(time.Nanosecond * 10)
		invoiceTracker.Stop()
	}()

	err = invoiceTracker.Start()
	assert.Equal(t, errors.Wrap(mockError, "could not check customer identity registration status").Error(), err.Error())
}

func Test_InvoiceTracker_Start_RefusesLargeFee(t *testing.T) {
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
		PaymentInfo:                dto.PaymentRate{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		AccountantID:               identity.FromAddress(acc.Address.Hex()),
		AccountantCaller:           &mockAccountantCaller{},
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		Publisher:                  &mockPublisher{},
		AccountantPromiseStorage:   accountantPromiseStorage,
		FeeProvider:                &mockTransactor{},
		MaxAllowedAccountantFee:    1500,
		BlockchainHelper:           &mockBlockchainHelper{feeToReturn: 1501, isRegistered: true},
	}
	invoiceTracker := NewInvoiceTracker(deps)

	go func() {
		time.Sleep(time.Nanosecond * 10)
		invoiceTracker.Stop()
	}()

	err = invoiceTracker.Start()
	assert.Equal(t, ErrAccountantFeeTooLarge, err)
}

func Test_InvoiceTracker_Start_BubblesAccountantCheckError(t *testing.T) {
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

	mockErr := errors.New("explosions everywhere")
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
		PaymentInfo:                dto.PaymentRate{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		AccountantID:               identity.FromAddress(acc.Address.Hex()),
		FeeProvider:                &mockTransactor{},
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		Publisher:                  &mockPublisher{},
		AccountantCaller:           &mockAccountantCaller{},
		AccountantPromiseStorage:   accountantPromiseStorage,
		BlockchainHelper:           &mockBlockchainHelper{errorToReturn: mockErr, isRegistered: true},
	}
	invoiceTracker := NewInvoiceTracker(deps)

	go func() {
		time.Sleep(time.Nanosecond * 10)
		invoiceTracker.Stop()
	}()

	err = invoiceTracker.Start()
	assert.Equal(t, errors.Wrap(mockErr, "could not get accountants fee").Error(), err.Error())
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
		ChargePeriod:               time.Millisecond,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		PaymentInfo:                dto.PaymentRate{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		AccountantID:               identity.FromAddress(acc.Address.Hex()),
		AccountantCaller:           &mockAccountantCaller{},
		AccountantPromiseStorage:   accountantPromiseStorage,
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		FeeProvider:                &mockTransactor{},
		Publisher:                  &mockPublisher{},
		BlockchainHelper:           &mockBlockchainHelper{isRegistered: true},
	}
	invoiceTracker := NewInvoiceTracker(deps)
	defer invoiceTracker.Stop()

	errChan := make(chan error)
	go func() { errChan <- invoiceTracker.Start() }()

	invoice := <-mockSender.chanToWriteTo
	b, err := hex.DecodeString(invoice.Hashlock)
	assert.NoError(t, err)
	exchangeMessageChan <- crypto.ExchangeMessage{
		Promise: crypto.Promise{
			Hashlock: b,
		},
	}

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
		PaymentInfo:                dto.PaymentRate{Price: money.NewMoney(1000000000000, money.CurrencyMyst), Duration: time.Minute},
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		AccountantID:               identity.FromAddress(acc.Address.Hex()),
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		AccountantCaller:           &mockAccountantCaller{},
		FeeProvider:                &mockTransactor{},
		AccountantPromiseStorage:   accountantPromiseStorage,
		BlockchainHelper:           &mockBlockchainHelper{isRegistered: true},
		Publisher:                  &mockPublisher{},
	}
	invoiceTracker := NewInvoiceTracker(deps)
	defer invoiceTracker.Stop()

	errChan := make(chan error)
	go func() { errChan <- invoiceTracker.Start() }()

	invoice := <-mockSender.chanToWriteTo
	assert.True(t, invoice.AgreementTotal > 0)
	assert.Len(t, invoice.Hashlock, 64)
	assert.Equal(t, strings.ToLower(acc.Address.Hex()), strings.ToLower(invoice.Provider))

	invoiceTracker.Stop()
	assert.NoError(t, <-errChan)
}

func Test_InvoiceTracker_FreeServiceSendsInvoices(t *testing.T) {
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
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		AccountantID:               identity.FromAddress(acc.Address.Hex()),
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		AccountantCaller:           &mockAccountantCaller{},
		FeeProvider:                &mockTransactor{},
		AccountantPromiseStorage:   accountantPromiseStorage,
		BlockchainHelper:           &mockBlockchainHelper{isRegistered: true},
		Publisher:                  &mockPublisher{},
	}
	invoiceTracker := NewInvoiceTracker(deps)
	defer invoiceTracker.Stop()

	errChan := make(chan error)
	go func() { errChan <- invoiceTracker.Start() }()

	invoice := <-mockSender.chanToWriteTo
	assert.Equal(t, uint64(0), invoice.AgreementTotal)
	assert.Len(t, invoice.Hashlock, 64)
	assert.Equal(t, strings.ToLower(acc.Address.Hex()), strings.ToLower(invoice.Provider))

	invoiceTracker.Stop()
	assert.NoError(t, <-errChan)
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
		addr, err := crypto.GenerateChannelAddress(acc.Address.Hex(), mockAccountantAddress, mockRegistryAddress, mockChannelImplementation)
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
	dir, err := ioutil.TempDir("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	msg1, addr1 := generateExchangeMessage(t, 10, crypto.Invoice{AgreementTotal: 10}, "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C")
	msg2, addr2 := generateExchangeMessage(t, 10, crypto.Invoice{AgreementTotal: 10, Hashlock: "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"}, "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C")
	msg3, addr3 := generateExchangeMessage(t, 10, crypto.Invoice{AgreementTotal: 10, Hashlock: "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"}, "")
	type fields struct {
		peer                       identity.Identity
		exchangeMessageChan        chan crypto.ExchangeMessage
		exchangeMessageWaitTimeout time.Duration
		accountantFailureCount     uint64
		accountantPromiseStorage   accountantPromiseStorage
		accountantID               identity.Identity
		AgreementID                uint64
		lastExchangeMessage        crypto.ExchangeMessage
		accountantCaller           accountantCaller
		invoicesSent               map[string]sentInvoice
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
				peer:                       identity.FromAddress(addr2),
			},
			em: &msg2,
		},
		{
			name:    "completes green path",
			wantErr: false,
			fields: fields{
				exchangeMessageWaitTimeout: time.Minute,
				exchangeMessageChan:        make(chan crypto.ExchangeMessage),
				accountantCaller:           &mockAccountantCaller{},
				accountantPromiseStorage:   &mockAccountantPromiseStorage{},
				peer:                       identity.FromAddress(addr3),
				registryAddress:            mockRegistryAddress,
				channelImplementation:      mockChannelImplementation,
				accountantID:               identity.FromAddress(mockAccountantAddress),
				invoicesSent: map[string]sentInvoice{
					hex.EncodeToString(msg3.Promise.Hashlock): sentInvoice{
						invoice: crypto.Invoice{
							Hashlock: hex.EncodeToString(msg3.Promise.Hashlock),
						},
					},
				},
			},
			em: &msg3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := InvoiceTrackerDeps{
				Peer:                       tt.fields.peer,
				ExchangeMessageChan:        tt.fields.exchangeMessageChan,
				ExchangeMessageWaitTimeout: tt.fields.exchangeMessageWaitTimeout,
				AccountantPromiseStorage:   tt.fields.accountantPromiseStorage,
				AccountantID:               tt.fields.accountantID,
				AccountantCaller:           tt.fields.accountantCaller,
				Registry:                   tt.fields.registryAddress,
				Publisher:                  &mockPublisher{},
				InvoiceStorage:             NewProviderInvoiceStorage(NewInvoiceStorage(bolt)),
				ChannelAddressCalculator:   NewChannelAddressCalculator(tt.fields.accountantID.Address, tt.fields.channelImplementation, tt.fields.registryAddress),
			}
			it := &InvoiceTracker{
				accountantFailureCount: tt.fields.accountantFailureCount,
				lastExchangeMessage:    tt.fields.lastExchangeMessage,
				agreementID:            tt.fields.AgreementID,
				deps:                   deps,
				invoicesSent:           tt.fields.invoicesSent,
			}
			if err := it.handleExchangeMessage(*tt.em); (err != nil) != tt.wantErr {
				t.Errorf("InvoiceTracker.receiveExchangeMessageOrTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockAccountantPromiseStorage struct {
}

func (maps *mockAccountantPromiseStorage) Store(providerID, accountantID identity.Identity, promise AccountantPromise) error {
	return nil
}

func (maps *mockAccountantPromiseStorage) Get(providerID, accountantID identity.Identity) (AccountantPromise, error) {
	return AccountantPromise{}, nil
}

type mockBlockchainHelper struct {
	feeToReturn   uint16
	errorToReturn error

	isRegistered      bool
	isRegisteredError error
}

func (mbh *mockBlockchainHelper) GetAccountantFee(accountantAddress common.Address) (uint16, error) {
	return mbh.feeToReturn, mbh.errorToReturn
}

func (mbh *mockBlockchainHelper) IsRegistered(registryAddress, addressToCheck common.Address) (bool, error) {
	return mbh.isRegistered, mbh.isRegisteredError
}

type event struct {
	name  string
	value interface{}
}

type mockPublisher struct {
	publicationChan chan event
}

func (mp *mockPublisher) Publish(topic string, payload interface{}) {
	if mp.publicationChan != nil {
		mp.publicationChan <- event{
			name:  topic,
			value: payload,
		}
	}
}
