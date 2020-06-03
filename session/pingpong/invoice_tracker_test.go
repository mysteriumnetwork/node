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
	stdErrors "errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/mbtime"
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

type mockAccountantCaller struct {
	errToReturn error
}

func (mac *mockAccountantCaller) RequestPromise(rp RequestPromise) (crypto.Promise, error) {
	return crypto.Promise{}, mac.errToReturn
}

func (mac *mockAccountantCaller) RevealR(r string, provider string, agreementID uint64) error {
	return mac.errToReturn
}

func Test_InvoiceTracker_Start_Stop(t *testing.T) {
	dir, err := ioutil.TempDir("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewMockKeystore()
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	mockSender := &MockPeerInvoiceSender{
		chanToWriteTo: make(chan crypto.Invoice, 10),
	}

	exchangeMessageChan := make(chan crypto.ExchangeMessage)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(mbtime.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	deps := InvoiceTrackerDeps{
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate:  market.PaymentRate{PerTime: time.Minute},
			},
		},
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		EventBus:                   mocks.NewEventBus(),
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ChargePeriodLeeway:         15 * time.Minute,
		ExchangeMessageChan:        exchangeMessageChan,
		FirstInvoiceSendDuration:   time.Nanosecond,
		FirstInvoiceSendTimeout:    time.Minute,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersAccountantID:      acc.Address,
		ProvidersAccountantID:      acc.Address,
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

func Test_InvoiceTracker_Start_RefusesLargeFee(t *testing.T) {
	dir, err := ioutil.TempDir("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewMockKeystore()
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	mockSender := &MockPeerInvoiceSender{
		chanToWriteTo: make(chan crypto.Invoice, 10),
	}

	exchangeMessageChan := make(chan crypto.ExchangeMessage)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(mbtime.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	deps := InvoiceTrackerDeps{
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate:  market.PaymentRate{PerTime: time.Minute},
			},
		},
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersAccountantID:      acc.Address,
		ProvidersAccountantID:      acc.Address,
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		EventBus:                   mocks.NewEventBus(),
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

	ks := identity.NewMockKeystore()
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
	tracker := session.NewTracker(mbtime.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	NewAccountantPromiseStorage(bolt)
	deps := InvoiceTrackerDeps{
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate:  market.PaymentRate{PerTime: time.Minute},
			},
		},
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ChargePeriodLeeway:         15 * time.Minute,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersAccountantID:      acc.Address,
		ProvidersAccountantID:      acc.Address,
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		EventBus:                   mocks.NewEventBus(),
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

	ks := identity.NewMockKeystore()
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	mockSender := &MockPeerInvoiceSender{
		chanToWriteTo: make(chan crypto.Invoice, 10),
	}

	exchangeMessageChan := make(chan crypto.ExchangeMessage)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(mbtime.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	deps := InvoiceTrackerDeps{
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate:  market.PaymentRate{PerTime: time.Minute},
			},
		},
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Millisecond,
		ChargePeriodLeeway:         15 * time.Minute,
		FirstInvoiceSendDuration:   time.Millisecond,
		FirstInvoiceSendTimeout:    time.Minute,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersAccountantID:      acc.Address,
		ProvidersAccountantID:      acc.Address,
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		EventBus:                   mocks.NewEventBus(),
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

	ks := identity.NewMockKeystore()
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)
	mockSender := &MockPeerInvoiceSender{
		chanToWriteTo: make(chan crypto.Invoice, 10),
	}

	exchangeMessageChan := make(chan crypto.ExchangeMessage)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(mbtime.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	deps := InvoiceTrackerDeps{
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(1000000000000, money.CurrencyMyst),
				rate:  market.PaymentRate{PerTime: time.Minute},
			},
		},
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ChargePeriodLeeway:         15 * time.Minute,
		FirstInvoiceSendDuration:   time.Nanosecond,
		FirstInvoiceSendTimeout:    time.Minute,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersAccountantID:      acc.Address,
		ProvidersAccountantID:      acc.Address,
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		BlockchainHelper:           &mockBlockchainHelper{isRegistered: true},
		EventBus:                   mocks.NewEventBus(),
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

func Test_InvoiceTracker_SendsFirstInvoice_Return_Timeout_Err(t *testing.T) {
	dir, err := ioutil.TempDir("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewMockKeystore()
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)
	mockSender := &MockPeerInvoiceSender{
		mockError: p2p.ErrSendTimeout,
	}

	exchangeMessageChan := make(chan crypto.ExchangeMessage)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(mbtime.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	deps := InvoiceTrackerDeps{
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(1000000000000, money.CurrencyMyst),
				rate:  market.PaymentRate{PerTime: time.Minute},
			},
		},
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ChargePeriodLeeway:         5 * time.Nanosecond,
		FirstInvoiceSendDuration:   time.Millisecond,
		FirstInvoiceSendTimeout:    time.Nanosecond,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersAccountantID:      acc.Address,
		ProvidersAccountantID:      acc.Address,
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		BlockchainHelper:           &mockBlockchainHelper{isRegistered: true},
		EventBus:                   mocks.NewEventBus(),
	}
	invoiceTracker := NewInvoiceTracker(deps)
	defer invoiceTracker.Stop()

	errChan := make(chan error)
	go func() { errChan <- invoiceTracker.Start() }()

	err = <-errChan

	if !stdErrors.Is(err, ErrFirstInvoiceSendTimeout) {
		t.Fatalf("expected err %v, got: %v", ErrFirstInvoiceSendTimeout, err)
	}
}

func Test_InvoiceTracker_FirstInvoice_Has_Static_Value(t *testing.T) {
	dir, err := ioutil.TempDir("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewMockKeystore()
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)
	mockSender := &MockPeerInvoiceSender{
		chanToWriteTo: make(chan crypto.Invoice, 10),
	}

	exchangeMessageChan := make(chan crypto.ExchangeMessage)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(mbtime.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	deps := InvoiceTrackerDeps{
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(1000000000000, money.CurrencyMyst),
				rate:  market.PaymentRate{PerTime: time.Minute},
			},
		},
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ChargePeriodLeeway:         15 * time.Minute,
		FirstInvoiceSendDuration:   time.Nanosecond,
		FirstInvoiceSendTimeout:    time.Minute,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersAccountantID:      acc.Address,
		ProvidersAccountantID:      acc.Address,
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		BlockchainHelper:           &mockBlockchainHelper{isRegistered: true},
		EventBus:                   mocks.NewEventBus(),
	}
	invoiceTracker := NewInvoiceTracker(deps)
	defer invoiceTracker.Stop()

	errChan := make(chan error)
	go func() { errChan <- invoiceTracker.Start() }()

	invoice := <-mockSender.chanToWriteTo
	assert.Equal(t, uint64(providerFirstInvoiceValue), invoice.AgreementTotal)
	assert.Len(t, invoice.Hashlock, 64)
	assert.Equal(t, strings.ToLower(acc.Address.Hex()), strings.ToLower(invoice.Provider))

	invoiceTracker.Stop()
	assert.NoError(t, <-errChan)
}

func Test_InvoiceTracker_FreeServiceSendsInvoices(t *testing.T) {
	dir, err := ioutil.TempDir("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewMockKeystore()
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)
	mockSender := &MockPeerInvoiceSender{
		chanToWriteTo: make(chan crypto.Invoice, 10),
	}

	exchangeMessageChan := make(chan crypto.ExchangeMessage)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(mbtime.Now)
	invoiceStorage := NewProviderInvoiceStorage(NewInvoiceStorage(bolt))
	deps := InvoiceTrackerDeps{
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ChargePeriodLeeway:         15 * time.Second,
		FirstInvoiceSendDuration:   time.Nanosecond,
		FirstInvoiceSendTimeout:    time.Minute,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersAccountantID:      acc.Address,
		ProvidersAccountantID:      acc.Address,
		ChannelAddressCalculator:   NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		BlockchainHelper:           &mockBlockchainHelper{isRegistered: true},
		EventBus:                   mocks.NewEventBus(),
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

func Test_sendsInvoiceIfThresholdReached(t *testing.T) {
	tracker := session.NewTracker(mbtime.Now)
	tracker.StartTracking()
	deps := InvoiceTrackerDeps{
		TimeTracker: &tracker,
		EventBus:    mocks.NewEventBus(),
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate: market.PaymentRate{
					PerTime: time.Minute,
					PerByte: 1,
				},
			},
		},
		MaxNotPaidInvoice: 100,
	}
	invoiceTracker := NewInvoiceTracker(deps)
	invoiceTracker.dataTransferred = DataTransferred{
		Up:   100,
		Down: 100,
	}
	invoiceTracker.invoiceDebounceRate = time.Nanosecond
	defer invoiceTracker.Stop()

	go invoiceTracker.sendInvoicesWhenNeeded(time.Millisecond * 5)

	res := <-invoiceTracker.invoiceChannel
	assert.True(t, res)
}

func Test_sendsInvoiceIfTimePassed(t *testing.T) {
	tracker := session.NewTracker(mbtime.Now)
	tracker.StartTracking()
	deps := InvoiceTrackerDeps{
		TimeTracker: &tracker,
		EventBus:    mocks.NewEventBus(),
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate: market.PaymentRate{
					PerTime: time.Minute,
					PerByte: 1,
				},
			},
		},
		MaxNotPaidInvoice: 100,
		ChargePeriod:      time.Millisecond,
	}
	invoiceTracker := NewInvoiceTracker(deps)
	invoiceTracker.dataTransferred = DataTransferred{
		Up:   1,
		Down: 1,
	}
	invoiceTracker.invoiceDebounceRate = time.Nanosecond
	defer invoiceTracker.Stop()

	go invoiceTracker.sendInvoicesWhenNeeded(time.Millisecond * 5)

	res := <-invoiceTracker.invoiceChannel
	assert.False(t, res)
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

	ks := identity.NewMockKeystore()
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
		accountantID               common.Address
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
				accountantID:               common.HexToAddress(mockAccountantAddress),
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
				ConsumersAccountantID:      tt.fields.accountantID,
				ProvidersAccountantID:      tt.fields.accountantID,
				Registry:                   tt.fields.registryAddress,
				EventBus:                   mocks.NewEventBus(),
				InvoiceStorage:             NewProviderInvoiceStorage(NewInvoiceStorage(bolt)),
				ChannelAddressCalculator:   NewChannelAddressCalculator(tt.fields.accountantID.Hex(), tt.fields.channelImplementation, tt.fields.registryAddress),
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

func Test_InvoiceTracker_RejectsInvalidAccountant(t *testing.T) {
	tracker := session.NewTracker(mbtime.Now)
	deps := InvoiceTrackerDeps{
		EventBus:              mocks.NewEventBus(),
		TimeTracker:           &tracker,
		ConsumersAccountantID: common.HexToAddress("0x1"),
		ProvidersAccountantID: common.HexToAddress("0x0"),
	}
	invoiceTracker := NewInvoiceTracker(deps)
	err := invoiceTracker.Start()
	assert.EqualError(t, err, fmt.Errorf("consumer wants to work with an unsupported accountant(%q) while provider expects %q", common.HexToAddress("0x1").Hex(), common.HexToAddress("0x0").Hex()).Error())
}

func TestInvoiceTracker_handleAccountantError(t *testing.T) {
	tests := []struct {
		name                      string
		maxAccountantFailureCount uint64
		err                       error
		wantErr                   error
	}{
		{
			name:    "ignores nil errors",
			wantErr: nil,
			err:     nil,
		},
		{
			name:    "handles wrapped errors",
			wantErr: ErrAccountantInternal,
			err:     errors.Wrap(ErrAccountantInternal, "pita bread"),
		},
		{
			name:    "bubbles internal on failure exceeded",
			wantErr: ErrAccountantInternal,
			err:     ErrAccountantInternal,
		},
		{
			name:                      "returns nil on internal not exceeding limit",
			wantErr:                   nil,
			maxAccountantFailureCount: 1,
			err:                       ErrAccountantInternal,
		},
		{
			name:    "bubbles hashlock missmatch on failure exceeded",
			wantErr: ErrAccountantHashlockMissmatch,
			err:     ErrAccountantHashlockMissmatch,
		},
		{
			name:                      "returns nil on hashlock missmatch not exceeding limit",
			wantErr:                   nil,
			maxAccountantFailureCount: 1,
			err:                       ErrAccountantHashlockMissmatch,
		},
		{
			name:                      "returns unknown error",
			wantErr:                   errors.New("unknown error"),
			maxAccountantFailureCount: 100,
			err:                       errors.New("unknown error"),
		},
		{
			name:                      "returns overspend",
			maxAccountantFailureCount: 100,
			wantErr:                   ErrAccountantOverspend,
			err:                       ErrAccountantOverspend,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := &InvoiceTracker{
				deps: InvoiceTrackerDeps{
					MaxAccountantFailureCount: tt.maxAccountantFailureCount,
				},
			}
			err := it.handleAccountantError(tt.err)
			if tt.wantErr == nil {
				assert.NoError(t, err, tt.name)
			} else {
				assert.EqualError(t, errors.Cause(err), tt.wantErr.Error(), tt.name)
			}
		})
	}
}

type mockPaymentMethod struct {
	price money.Money
	t     string
	rate  market.PaymentRate
}

// Service price per unit of metering
func (mpm *mockPaymentMethod) GetPrice() money.Money {
	return mpm.price
}

func (mpm *mockPaymentMethod) GetType() string {
	return mpm.t
}

func (mpm *mockPaymentMethod) GetRate() market.PaymentRate {
	return mpm.rate
}

type mockEncryptor struct {
	errToReturn error
}

func (me *mockEncryptor) Decrypt(addr common.Address, encrypted []byte) ([]byte, error) {
	return encrypted, me.errToReturn
}

func (me *mockEncryptor) Encrypt(addr common.Address, plaintext []byte) ([]byte, error) {
	return plaintext, me.errToReturn
}

type mockAccountantPromiseStorage struct {
	toReturn    AccountantPromise
	errToReturn error
}

func (maps *mockAccountantPromiseStorage) Store(_ identity.Identity, _ common.Address, _ AccountantPromise) error {
	return maps.errToReturn
}

func (maps *mockAccountantPromiseStorage) Get(_ identity.Identity, _ common.Address) (AccountantPromise, error) {
	return maps.toReturn, maps.errToReturn
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

type testEvent struct {
	name  string
	value interface{}
}

type mockPublisher struct {
	publicationChan chan testEvent
}

func (mp *mockPublisher) Publish(topic string, payload interface{}) {
	if mp.publicationChan != nil {
		mp.publicationChan <- testEvent{
			name:  topic,
			value: payload,
		}
	}
}

func (mp *mockPublisher) Subscribe(topic string, fn interface{}) error {
	return nil
}
func (mp *mockPublisher) SubscribeAsync(topic string, fn interface{}) error {
	return nil
}

func (mp *mockPublisher) Unsubscribe(topic string, fn interface{}) error {
	return nil
}
