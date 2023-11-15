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
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/mbtime"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/mysteriumnetwork/payments/observer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	mockRegistryAddress       = "0xE6b3a5c92e7c1f9543A0aEE9A93fE2F6B584c1f7"
	mockHermesAddress         = "0xf28DB7aDf64A2811202B149aa4733A1FB9100e5c"
	mockChannelImplementation = "0xa26b684d8dBa935DD34544FBd3Ab4d7FDe1C4D07"
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

type mockHermesCaller struct {
	errToReturn error
}

func (mac *mockHermesCaller) RequestPromise(rp RequestPromise) (crypto.Promise, error) {
	return crypto.Promise{}, mac.errToReturn
}

func (mac *mockHermesCaller) PayAndSettle(rp RequestPromise) (crypto.Promise, error) {
	return crypto.Promise{}, mac.errToReturn
}

func (mac *mockHermesCaller) RevealR(r string, provider string, agreementID *big.Int) error {
	return mac.errToReturn
}

func (mac *mockHermesCaller) UpdatePromiseFee(promise crypto.Promise, newFee *big.Int) (crypto.Promise, error) {
	return promise, nil
}

func (mac *mockHermesCaller) GetConsumerData(chainID int64, id string, cacheTime time.Duration) (HermesUserInfo, error) {
	return HermesUserInfo{}, nil
}

func (mac *mockHermesCaller) GetProviderData(chainID int64, id string) (HermesUserInfo, error) {
	return HermesUserInfo{}, nil
}

func (mac *mockHermesCaller) SyncProviderPromise(promise crypto.Promise, signer identity.Signer) error {
	return nil
}

func (mac *mockHermesCaller) RefreshLatestProviderPromise(chainID int64, id string, hashlock, recoveryData []byte, signer identity.Signer) (crypto.Promise, error) {
	return crypto.Promise{}, nil
}

func Test_InvoiceTracker_Start_Stop(t *testing.T) {
	dir, err := os.MkdirTemp("", "invoice_tracker_test")
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
		AgreedPrice:                *market.NewPrice(600, 0),
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		EventBus:                   mocks.NewEventBus(),
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ChargePeriodLeeway:         15 * time.Minute,
		LimitChargePeriod:          time.Nanosecond,
		LimitNotPaidInvoice:        big.NewInt(0),
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersHermesID:          acc.Address,
		AddressProvider:            &mockAddressProvider{},
		HermesStatusChecker:        &mockHermesStatusChecker{statusToReturn: HermesStatus{IsActive: true}},
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
	dir, err := os.MkdirTemp("", "invoice_tracker_test")
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
		AgreedPrice:                *market.NewPrice(600, 0),
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		LimitChargePeriod:          time.Nanosecond,
		LimitNotPaidInvoice:        big.NewInt(0),
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersHermesID:          acc.Address,
		AddressProvider:            &mockAddressProvider{},
		EventBus:                   mocks.NewEventBus(),
		MaxAllowedHermesFee:        1500,
		HermesStatusChecker:        &mockHermesStatusChecker{statusToReturn: HermesStatus{IsActive: true, Fee: 1501}},
	}
	invoiceTracker := NewInvoiceTracker(deps)

	go func() {
		time.Sleep(time.Nanosecond * 10)
		invoiceTracker.Stop()
	}()

	err = invoiceTracker.Start()
	assert.Equal(t, ErrHermesFeeTooLarge, err)
}

func Test_InvoiceTracker_Start_BubblesHermesCheckError(t *testing.T) {
	dir, err := os.MkdirTemp("", "invoice_tracker_test")
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
	NewHermesPromiseStorage(bolt)
	deps := InvoiceTrackerDeps{
		AgreedPrice:                *market.NewPrice(600, 0),
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		ChargePeriod:               time.Nanosecond,
		ChargePeriodLeeway:         15 * time.Minute,
		LimitChargePeriod:          time.Nanosecond,
		LimitNotPaidInvoice:        big.NewInt(0),
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersHermesID:          acc.Address,
		AddressProvider:            &mockAddressProvider{},
		EventBus:                   mocks.NewEventBus(),
		HermesStatusChecker:        &mockHermesStatusChecker{errToReturn: mockErr},
	}
	invoiceTracker := NewInvoiceTracker(deps)

	go func() {
		time.Sleep(time.Nanosecond * 10)
		invoiceTracker.Stop()
	}()

	err = invoiceTracker.Start()
	assert.Equal(t, errors.Wrap(mockErr, "could not check hermes status").Error(), err.Error())
}

func Test_InvoiceTracker_BubblesErrors(t *testing.T) {
	dir, err := os.MkdirTemp("", "invoice_tracker_test")
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
		AgreedPrice:                *market.NewPrice(600, 0),
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		LimitChargePeriod:          time.Nanosecond,
		LimitNotPaidInvoice:        big.NewInt(0),
		ChargePeriod:               time.Millisecond,
		ChargePeriodLeeway:         15 * time.Minute,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersHermesID:          acc.Address,
		AddressProvider:            &mockAddressProvider{},
		EventBus:                   mocks.NewEventBus(),
		HermesStatusChecker:        &mockHermesStatusChecker{statusToReturn: HermesStatus{IsActive: true}},
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
	dir, err := os.MkdirTemp("", "invoice_tracker_test")
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
		AgreedPrice:                *market.NewPrice(60000000000000, 0),
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		LimitChargePeriod:          time.Nanosecond,
		ChargePeriod:               time.Nanosecond,
		LimitNotPaidInvoice:        big.NewInt(0),
		MaxNotPaidInvoice:          big.NewInt(0),
		ChargePeriodLeeway:         15 * time.Minute,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersHermesID:          acc.Address,
		AddressProvider:            &mockAddressProvider{},
		HermesStatusChecker:        &mockHermesStatusChecker{statusToReturn: HermesStatus{IsActive: true}},
		EventBus:                   mocks.NewEventBus(),
	}
	invoiceTracker := NewInvoiceTracker(deps)
	defer invoiceTracker.Stop()

	errChan := make(chan error)
	go func() { errChan <- invoiceTracker.Start() }()

	invoice := <-mockSender.chanToWriteTo
	assert.True(t, invoice.AgreementTotal.Cmp(new(big.Int)) > 0)
	assert.Len(t, invoice.Hashlock, 64)
	assert.Equal(t, strings.ToLower(acc.Address.Hex()), strings.ToLower(invoice.Provider))

	invoiceTracker.Stop()
	assert.NoError(t, <-errChan)
}

func Test_InvoiceTracker_FirstInvoice_Has_Static_Value(t *testing.T) {
	dir, err := os.MkdirTemp("", "invoice_tracker_test")
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
		AgreedPrice:                *market.NewPrice(60000000000000, 0),
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		LimitChargePeriod:          time.Nanosecond,
		LimitNotPaidInvoice:        big.NewInt(0),
		ChargePeriod:               time.Nanosecond,
		ChargePeriodLeeway:         15 * time.Minute,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersHermesID:          acc.Address,
		AddressProvider:            &mockAddressProvider{},
		HermesStatusChecker:        &mockHermesStatusChecker{statusToReturn: HermesStatus{IsActive: true}},
		EventBus:                   mocks.NewEventBus(),
	}
	invoiceTracker := NewInvoiceTracker(deps)
	defer invoiceTracker.Stop()

	errChan := make(chan error)
	go func() { errChan <- invoiceTracker.Start() }()

	invoice := <-mockSender.chanToWriteTo
	assert.Equal(t, providerFirstInvoiceValue, invoice.AgreementTotal)
	assert.Len(t, invoice.Hashlock, 64)
	assert.Equal(t, strings.ToLower(acc.Address.Hex()), strings.ToLower(invoice.Provider))

	invoiceTracker.Stop()
	assert.NoError(t, <-errChan)
}

func Test_InvoiceTracker_FreeServiceSendsInvoices(t *testing.T) {
	dir, err := os.MkdirTemp("", "invoice_tracker_test")
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
		AgreedPrice:                *market.NewPrice(600, 0),
		Peer:                       identity.FromAddress("some peer"),
		PeerInvoiceSender:          mockSender,
		InvoiceStorage:             invoiceStorage,
		TimeTracker:                &tracker,
		LimitChargePeriod:          time.Nanosecond,
		LimitNotPaidInvoice:        big.NewInt(0),
		ChargePeriod:               time.Nanosecond,
		ChargePeriodLeeway:         15 * time.Second,
		ExchangeMessageChan:        exchangeMessageChan,
		ExchangeMessageWaitTimeout: time.Second,
		ProviderID:                 identity.FromAddress(acc.Address.Hex()),
		ConsumersHermesID:          acc.Address,
		AddressProvider:            &mockAddressProvider{},
		HermesStatusChecker:        &mockHermesStatusChecker{statusToReturn: HermesStatus{IsActive: true}},
		EventBus:                   mocks.NewEventBus(),
	}
	invoiceTracker := NewInvoiceTracker(deps)
	defer invoiceTracker.Stop()

	errChan := make(chan error)
	go func() { errChan <- invoiceTracker.Start() }()

	invoice := <-mockSender.chanToWriteTo
	assert.Equal(t, big.NewInt(0), invoice.AgreementTotal)
	assert.Len(t, invoice.Hashlock, 64)
	assert.Equal(t, strings.ToLower(acc.Address.Hex()), strings.ToLower(invoice.Provider))

	invoiceTracker.Stop()
	assert.NoError(t, <-errChan)
}

func Test_sendsInvoiceIfThresholdReached(t *testing.T) {
	tracker := session.NewTracker(mbtime.Now)
	tracker.StartTracking()
	deps := InvoiceTrackerDeps{
		TimeTracker:       &tracker,
		EventBus:          mocks.NewEventBus(),
		AgreedPrice:       *market.NewPrice(600, 10995116277760),
		MaxNotPaidInvoice: big.NewInt(100),
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
		TimeTracker:         &tracker,
		EventBus:            mocks.NewEventBus(),
		AgreedPrice:         *market.NewPrice(600, 0),
		MaxNotPaidInvoice:   big.NewInt(100),
		ChargePeriod:        time.Millisecond * 2,
		LimitChargePeriod:   time.Millisecond * 3,
		LimitNotPaidInvoice: big.NewInt(0),
	}
	invoiceTracker := NewInvoiceTracker(deps)
	invoiceTracker.dataTransferred = DataTransferred{
		Up:   1,
		Down: 1,
	}
	invoiceTracker.invoiceDebounceRate = time.Nanosecond

	wait := make(chan struct{}, 0)
	go func() {
		defer close(wait)
		invoiceTracker.sendInvoicesWhenNeeded(time.Millisecond * 5)
	}()

	res := <-invoiceTracker.invoiceChannel
	assert.False(t, res)
	res = <-invoiceTracker.invoiceChannel
	assert.False(t, res)
	invoiceTracker.Stop()

	<-wait
	// Test that MaxUnpaid and ChargePeriod Increased
	assert.Equal(t, time.Millisecond*3, invoiceTracker.deps.ChargePeriod, "charge period should increase up to limit")

}

func Test_sendsInvoiceIfDataUsed(t *testing.T) {
	tracker := session.NewTracker(mbtime.Now)
	tracker.StartTracking()
	deps := InvoiceTrackerDeps{
		TimeTracker:         &tracker,
		EventBus:            mocks.NewEventBus(),
		AgreedPrice:         *market.NewPrice(600, 100),
		MaxNotPaidInvoice:   big.NewInt(100),
		ChargePeriod:        time.Millisecond * 2,
		LimitChargePeriod:   time.Millisecond * 3,
		LimitNotPaidInvoice: big.NewInt(140),
	}
	invoiceTracker := NewInvoiceTracker(deps)
	invoiceTracker.invoiceDebounceRate = time.Nanosecond
	invoiceTracker.dataTransferred = DataTransferred{
		Up:   1000000000,
		Down: 1000000000,
	}
	invoiceTracker.invoiceDebounceRate = time.Millisecond * 1

	wait := make(chan struct{}, 0)
	go func() {
		defer close(wait)
		invoiceTracker.sendInvoicesWhenNeeded(time.Millisecond * 5)
	}()

	res := <-invoiceTracker.invoiceChannel
	assert.True(t, res)
	res = <-invoiceTracker.invoiceChannel
	assert.True(t, res)
	invoiceTracker.Stop()

	<-wait
	// Test that MaxUnpaid and ChargePeriod Increased
	assert.Equal(t, big.NewInt(140), invoiceTracker.deps.MaxNotPaidInvoice, "max unpaid invoice should increase up to limit")

}

func Test_calculateMaxNotReceivedExchangeMessageCount(t *testing.T) {
	res := calculateMaxNotReceivedExchangeMessageCount(time.Minute*5, time.Second*240)
	assert.Equal(t, uint64(1), res)
	res = calculateMaxNotReceivedExchangeMessageCount(time.Minute*5, time.Second*20)
	assert.Equal(t, uint64(15), res)
	res = calculateMaxNotReceivedExchangeMessageCount(time.Hour*2, time.Second*20)
	assert.Equal(t, uint64(360), res)
}

func generateExchangeMessage(t *testing.T, amount *big.Int, invoice crypto.Invoice, channel string) (crypto.ExchangeMessage, string) {
	dir, err := os.MkdirTemp("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewMockKeystore()
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	err = ks.Unlock(acc, "")
	assert.Nil(t, err)

	if channel == "" {
		addr, err := crypto.GenerateChannelAddress(acc.Address.Hex(), mockHermesAddress, mockRegistryAddress, mockChannelImplementation)
		assert.Nil(t, err)
		channel = addr
	}

	em, err := crypto.CreateExchangeMessage(1, invoice, amount, channel, "", ks, acc.Address)
	assert.Nil(t, err)
	if em != nil {
		return *em, acc.Address.Hex()
	}
	return crypto.ExchangeMessage{}, acc.Address.Hex()
}

func TestInvoiceTracker_receiveExchangeMessageOrTimeout(t *testing.T) {
	dir, err := os.MkdirTemp("", "invoice_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	msg1, addr1 := generateExchangeMessage(t, big.NewInt(10), crypto.Invoice{AgreementTotal: big.NewInt(10), AgreementID: new(big.Int), TransactorFee: new(big.Int)}, "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C")
	msg2, addr2 := generateExchangeMessage(t, big.NewInt(10), crypto.Invoice{AgreementTotal: big.NewInt(10), AgreementID: new(big.Int), TransactorFee: new(big.Int), Hashlock: "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"}, "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C")
	msg3, addr3 := generateExchangeMessage(t, big.NewInt(10), crypto.Invoice{AgreementTotal: big.NewInt(10), AgreementID: new(big.Int), TransactorFee: new(big.Int), Hashlock: "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"}, "")
	type fields struct {
		peer                       identity.Identity
		exchangeMessageChan        chan crypto.ExchangeMessage
		exchangeMessageWaitTimeout time.Duration
		hermesFailureCount         uint64
		hermesPromiseStorage       hermesPromiseStorage
		hermesID                   common.Address
		AgreementID                *big.Int
		lastExchangeMessage        crypto.ExchangeMessage
		invoicesSent               map[string]sentInvoice
		addressProvider            addressProvider
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
				lastExchangeMessage: crypto.ExchangeMessage{
					Promise: crypto.Promise{
						Amount: new(big.Int),
						Fee:    new(big.Int),
					},
					AgreementID:    new(big.Int),
					AgreementTotal: new(big.Int),
				},
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
				lastExchangeMessage: crypto.ExchangeMessage{
					Promise: crypto.Promise{
						Amount: new(big.Int),
						Fee:    new(big.Int),
					},
					AgreementID:    new(big.Int),
					AgreementTotal: new(big.Int),
				},
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
				lastExchangeMessage: crypto.ExchangeMessage{
					Promise: crypto.Promise{
						Amount: new(big.Int),
						Fee:    new(big.Int),
					},
					AgreementID:    new(big.Int),
					AgreementTotal: new(big.Int),
				},
			},
			em: &msg2,
		},
		{
			name:    "completes green path",
			wantErr: false,
			fields: fields{
				addressProvider:            &mockAddressProvider{addrToReturn: common.BytesToAddress(msg3.Promise.ChannelID)},
				exchangeMessageWaitTimeout: time.Minute,
				exchangeMessageChan:        make(chan crypto.ExchangeMessage),
				hermesPromiseStorage:       &mockHermesPromiseStorage{},
				peer:                       identity.FromAddress(addr3),
				hermesID:                   common.HexToAddress(mockHermesAddress),
				lastExchangeMessage: crypto.ExchangeMessage{
					Promise: crypto.Promise{
						Amount: new(big.Int),
						Fee:    new(big.Int),
					},
					AgreementID:    new(big.Int),
					AgreementTotal: new(big.Int),
				},
				invoicesSent: map[string]sentInvoice{
					hex.EncodeToString(msg3.Promise.Hashlock): {
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
				ConsumersHermesID:          tt.fields.hermesID,
				EventBus:                   mocks.NewEventBus(),
				InvoiceStorage:             NewProviderInvoiceStorage(NewInvoiceStorage(bolt)),
				AddressProvider:            tt.fields.addressProvider,
				AgreedPrice:                *market.NewPrice(0, 0),
				LimitChargePeriod:          time.Nanosecond,
				LimitNotPaidInvoice:        big.NewInt(0),
			}
			it := &InvoiceTracker{
				hermesFailureCount:  tt.fields.hermesFailureCount,
				lastExchangeMessage: tt.fields.lastExchangeMessage,
				agreementID:         tt.fields.AgreementID,
				deps:                deps,
				invoicesSent:        tt.fields.invoicesSent,
			}
			if err := it.handleExchangeMessage(*tt.em); (err != nil) != tt.wantErr {
				t.Errorf("InvoiceTracker.receiveExchangeMessageOrTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInvoiceTracker_handleHermesError(t *testing.T) {
	tests := []struct {
		name                  string
		maxHermesFailureCount uint64
		err                   error
		wantErr               error
	}{
		{
			name:    "ignores nil errors",
			wantErr: nil,
			err:     nil,
		},
		{
			name:    "handles wrapped errors",
			wantErr: ErrHermesInternal,
			err:     errors.Wrap(ErrHermesInternal, "pita bread"),
		},
		{
			name:    "bubbles internal on failure exceeded",
			wantErr: ErrHermesInternal,
			err:     ErrHermesInternal,
		},
		{
			name:                  "returns nil on internal not exceeding limit",
			wantErr:               nil,
			maxHermesFailureCount: 1,
			err:                   ErrHermesInternal,
		},
		{
			name:    "bubbles hashlock missmatch on failure exceeded",
			wantErr: ErrHermesHashlockMissmatch,
			err:     ErrHermesHashlockMissmatch,
		},
		{
			name:                  "returns nil on hashlock missmatch not exceeding limit",
			wantErr:               nil,
			maxHermesFailureCount: 1,
			err:                   ErrHermesHashlockMissmatch,
		},
		{
			name:    "returns unknown error if failures exceeded",
			wantErr: errors.New("unknown error"),
			err:     errors.New("unknown error"),
		},
		{
			name:                  "returns nil on unknown error if failures not exceeded",
			wantErr:               nil,
			maxHermesFailureCount: 100,
			err:                   errors.New("unknown error"),
		},
		{
			name:                  "returns overspend",
			maxHermesFailureCount: 100,
			wantErr:               ErrHermesOverspend,
			err:                   ErrHermesOverspend,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := &InvoiceTracker{
				deps: InvoiceTrackerDeps{
					MaxHermesFailureCount: tt.maxHermesFailureCount,
					LimitChargePeriod:     time.Nanosecond,
					LimitNotPaidInvoice:   big.NewInt(0),
				},
			}
			err := it.handleHermesError(tt.err)
			if tt.wantErr == nil {
				assert.NoError(t, err, tt.name)
			} else {
				assert.EqualError(t, errors.Cause(err), tt.wantErr.Error(), tt.name)
			}
		})
	}
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

type mockHermesPromiseStorage struct {
	toReturn    HermesPromise
	errToReturn error
}

func (maps *mockHermesPromiseStorage) Store(_ HermesPromise) error {
	return maps.errToReturn
}

func (maps *mockHermesPromiseStorage) Delete(_ HermesPromise) error {
	return maps.errToReturn
}

func (maps *mockHermesPromiseStorage) Get(chainID int64, _ string) (HermesPromise, error) {
	return maps.toReturn, maps.errToReturn
}

func (maps *mockHermesPromiseStorage) List(_ HermesPromiseFilter) ([]HermesPromise, error) {
	return []HermesPromise{maps.toReturn}, maps.errToReturn
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

func (mp *mockPublisher) SubscribeWithUID(topic, uid string, fn interface{}) error {
	return nil
}

func (mp *mockPublisher) SubscribeAsync(topic string, fn interface{}) error {
	return nil
}

func (mp *mockPublisher) Unsubscribe(topic string, fn interface{}) error {
	return nil
}

func (mp *mockPublisher) UnsubscribeWithUID(topic, uid string, fn interface{}) error {
	return nil
}

type mockObserver struct {
}

func (mo *mockObserver) GetHermeses(f *observer.HermesFilter) (observer.HermesesResponse, error) {
	return observer.HermesesResponse{}, nil
}

func (mo *mockObserver) GetHermesData(chainId int64, hermesAddress common.Address) (*observer.HermesResponse, error) {
	return nil, nil
}

func TestInvoiceTracker_validateExchangeMessage(t *testing.T) {
	type fields struct {
		deps InvoiceTrackerDeps
	}
	type args struct {
		em crypto.ExchangeMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			args: args{
				em: crypto.ExchangeMessage{
					HermesID: "0x1",
				},
			},
			name:    "rejects exchange message with unsupported hermes",
			wantErr: true,
			fields: fields{
				deps: InvoiceTrackerDeps{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := &InvoiceTracker{
				deps: tt.fields.deps,
			}
			if err := it.validateExchangeMessage(tt.args.em); (err != nil) != tt.wantErr {
				t.Errorf("InvoiceTracker.validateExchangeMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockHermesStatusChecker struct {
	statusToReturn HermesStatus
	errToReturn    error
}

func (mhsc *mockHermesStatusChecker) GetHermesStatus(chainID int64, registryAddress common.Address, hermesID common.Address) (HermesStatus, error) {
	return mhsc.statusToReturn, mhsc.errToReturn
}
