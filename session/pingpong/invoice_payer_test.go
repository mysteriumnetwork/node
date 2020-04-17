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
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/mbtime"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type MockPeerExchangeMessageSender struct {
	mockError     error
	chanToWriteTo chan crypto.ExchangeMessage
}

func (mpems *MockPeerExchangeMessageSender) Send(em crypto.ExchangeMessage) error {
	if mpems.chanToWriteTo != nil {
		mpems.chanToWriteTo <- em
	}
	return mpems.mockError
}

func Test_InvoicePayer_Start_Stop(t *testing.T) {
	dir, err := ioutil.TempDir("", "exchange_message_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	mockSender := &MockPeerExchangeMessageSender{
		chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
	}

	invoiceChan := make(chan crypto.Invoice)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(mbtime.Now)
	totalsStorage := NewConsumerTotalsStorage(bolt, eventbus.New())
	deps := InvoicePayerDeps{
		InvoiceChan:               invoiceChan,
		PeerExchangeMessageSender: mockSender,
		ConsumerTotalsStorage:     totalsStorage,
		TimeTracker:               &tracker,
		Ks:                        ks,
		ChannelAddressCalculator:  NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		Identity:                  identity.FromAddress(acc.Address.Hex()),
		Peer:                      identity.FromAddress("0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"),
		EventBus:                  mocks.NewEventBus(),
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate:  market.PaymentRate{PerTime: time.Minute},
			},
		},
	}
	InvoicePayer := NewInvoicePayer(deps)

	go func() {
		time.Sleep(time.Nanosecond * 10)
		InvoicePayer.Stop()
	}()

	err = InvoicePayer.Start()
	assert.Nil(t, err)
}

func Test_InvoicePayer_SendsMessage(t *testing.T) {
	dir, err := ioutil.TempDir("", "exchange_message_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	err = ks.Unlock(acc, "")
	assert.Nil(t, err)

	mockSender := &MockPeerExchangeMessageSender{
		chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
	}

	invoiceChan := make(chan crypto.Invoice)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(mbtime.Now)
	totalsStorage := NewConsumerTotalsStorage(bolt, eventbus.New())
	totalsStorage.Store(identity.FromAddress(acc.Address.Hex()), common.Address{}, 10)
	deps := InvoicePayerDeps{
		InvoiceChan:               invoiceChan,
		PeerExchangeMessageSender: mockSender,
		ConsumerTotalsStorage:     totalsStorage,
		TimeTracker:               &tracker,
		EventBus:                  mocks.NewEventBus(),
		Ks:                        ks,
		ChannelAddressCalculator:  NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		Identity:                  identity.FromAddress(acc.Address.Hex()),
		Peer:                      identity.FromAddress("0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"),
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate:  market.PaymentRate{PerTime: time.Minute},
			},
		},
	}
	InvoicePayer := NewInvoicePayer(deps)

	mockInvoice := crypto.Invoice{
		AgreementID:    1,
		AgreementTotal: 0,
		TransactorFee:  0,
		Hashlock:       "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C",
		Provider:       deps.Peer.Address,
	}

	testDone := make(chan struct{})

	defer InvoicePayer.Stop()
	go func() {
		err := InvoicePayer.Start()
		assert.Nil(t, err)
		testDone <- struct{}{}
	}()

	invoiceChan <- mockInvoice

	exchangeMessage := <-mockSender.chanToWriteTo
	InvoicePayer.Stop()

	addr, err := exchangeMessage.RecoverConsumerIdentity()
	assert.Nil(t, err)
	assert.Equal(t, acc.Address.Hex(), addr.Hex())
	assert.Equal(t, uint64(10), exchangeMessage.Promise.Amount)

	<-testDone
}

func Test_InvoicePayer_SendsMessage_OnFreeService(t *testing.T) {
	dir, err := ioutil.TempDir("", "exchange_message_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	err = ks.Unlock(acc, "")
	assert.Nil(t, err)

	mockSender := &MockPeerExchangeMessageSender{
		chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
	}

	invoiceChan := make(chan crypto.Invoice)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(mbtime.Now)
	totalsStorage := NewConsumerTotalsStorage(bolt, eventbus.New())
	totalsStorage.Store(identity.FromAddress(acc.Address.Hex()), common.Address{}, 0)
	deps := InvoicePayerDeps{
		InvoiceChan:               invoiceChan,
		PeerExchangeMessageSender: mockSender,
		ConsumerTotalsStorage:     totalsStorage,
		TimeTracker:               &tracker,
		EventBus:                  mocks.NewEventBus(),
		Ks:                        ks,
		ChannelAddressCalculator:  NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		Identity:                  identity.FromAddress(acc.Address.Hex()),
		Peer:                      identity.FromAddress("0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"),
	}
	InvoicePayer := NewInvoicePayer(deps)

	mockInvoice := crypto.Invoice{
		AgreementID:    1,
		AgreementTotal: 0,
		TransactorFee:  0,
		Hashlock:       "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C",
		Provider:       deps.Peer.Address,
	}

	testDone := make(chan struct{})

	defer InvoicePayer.Stop()
	go func() {
		err := InvoicePayer.Start()
		assert.Nil(t, err)
		testDone <- struct{}{}
	}()

	invoiceChan <- mockInvoice

	exchangeMessage := <-mockSender.chanToWriteTo
	InvoicePayer.Stop()
	addr, err := exchangeMessage.RecoverConsumerIdentity()
	assert.Nil(t, err)

	assert.Equal(t, acc.Address.Hex(), addr.Hex())

	<-testDone
}

func Test_InvoicePayer_BubblesErrors(t *testing.T) {
	dir, err := ioutil.TempDir("", "exchange_message_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	mockSender := &MockPeerExchangeMessageSender{
		chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
	}

	invoiceChan := make(chan crypto.Invoice)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(mbtime.Now)
	totalsStorage := NewConsumerTotalsStorage(bolt, eventbus.New())
	deps := InvoicePayerDeps{
		InvoiceChan:               invoiceChan,
		EventBus:                  mocks.NewEventBus(),
		PeerExchangeMessageSender: mockSender,
		ConsumerTotalsStorage:     totalsStorage,
		TimeTracker:               &tracker,
		Ks:                        ks,
		ChannelAddressCalculator:  NewChannelAddressCalculator(acc.Address.Hex(), acc.Address.Hex(), acc.Address.Hex()),
		Identity:                  identity.FromAddress(acc.Address.Hex()),
		Peer:                      identity.FromAddress("0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"),
		Proposal: market.ServiceProposal{
			PaymentMethod: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate:  market.PaymentRate{PerTime: time.Minute},
			},
		},
	}
	InvoicePayer := NewInvoicePayer(deps)
	defer InvoicePayer.Stop()
	errChan := make(chan error)
	go func() { errChan <- InvoicePayer.Start() }()

	invoiceChan <- crypto.Invoice{}

	err = <-errChan
	assert.Error(t, err)
}

func TestInvoicePayer_isInvoiceOK(t *testing.T) {
	type fields struct {
		peer        identity.Identity
		timeTracker timeTracker
		proposal    market.ServiceProposal
	}
	tests := []struct {
		name    string
		fields  fields
		invoice crypto.Invoice
		wantErr bool
	}{
		{
			name: "errors on invalid peer id",
			fields: fields{
				peer: identity.FromAddress("0x01"),
			},
			invoice: crypto.Invoice{
				Provider: "0x02",
			},
			wantErr: true,
		},
		{
			name: "errors on too large invoice",
			fields: fields{
				peer: identity.FromAddress("0x01"),
				timeTracker: &mockTimeTracker{
					timeToReturn: time.Minute,
				},
				proposal: market.ServiceProposal{
					PaymentMethod: &mockPaymentMethod{
						price: money.NewMoney(100000, money.CurrencyMyst),
						rate:  market.PaymentRate{PerTime: time.Minute},
					},
				},
			},
			invoice: crypto.Invoice{
				TransactorFee:  0,
				AgreementID:    1,
				AgreementTotal: 150100,
				Provider:       "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C",
			},
			wantErr: true,
		},
		{
			name: "accepts proper invoice",
			fields: fields{
				peer: identity.FromAddress("0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"),
				timeTracker: &mockTimeTracker{
					timeToReturn: time.Minute,
				},
				proposal: market.ServiceProposal{
					PaymentMethod: &mockPaymentMethod{
						price: money.NewMoney(100000, money.CurrencyMyst),
						rate:  market.PaymentRate{PerTime: time.Minute},
					},
				},
			},
			invoice: crypto.Invoice{
				TransactorFee:  0,
				AgreementID:    1,
				AgreementTotal: 100000,
				Provider:       "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emt := &InvoicePayer{
				deps: InvoicePayerDeps{
					TimeTracker: tt.fields.timeTracker,
					Proposal:    tt.fields.proposal,
					Peer:        tt.fields.peer,
				},
			}
			if err := emt.isInvoiceOK(tt.invoice); (err != nil) != tt.wantErr {
				t.Errorf("InvoicePayer.isInvoiceOK() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInvoicePayer_incrementGrandTotalPromised(t *testing.T) {
	type fields struct {
		consumerTotalsStorage *mockConsumerTotalsStorage
	}
	type args struct {
		amount uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    uint64
	}{
		{
			name: "returns the error from storage",
			fields: fields{
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					bus: eventbus.New(),
					err: errors.New("some error"),
				},
			},
			wantErr: true,
		},
		{
			name: "adds to zero if not found",
			fields: fields{
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					bus: eventbus.New(),
					err: ErrNotFound,
				},
			},
			args: args{
				amount: 11,
			},
			wantErr: false,
			want:    11,
		},
		{
			name: "adds to value if found",
			fields: fields{
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					bus: eventbus.New(),
					res: 15,
				},
			},
			args: args{
				amount: 11,
			},
			wantErr: false,
			want:    26,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emt := &InvoicePayer{
				deps: InvoicePayerDeps{
					ConsumerTotalsStorage: tt.fields.consumerTotalsStorage,
				},
			}
			if err := emt.incrementGrandTotalPromised(tt.args.amount); (err != nil) != tt.wantErr {
				t.Errorf("InvoicePayer.incrementGrandTotalPromised() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := tt.fields.consumerTotalsStorage.calledWith
			if got != tt.want {
				t.Errorf("InvoicePayer.incrementGrandTotalPromised() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvoicePayer_calculateAmountToPromise(t *testing.T) {
	type fields struct {
		peer                  identity.Identity
		lastInvoice           crypto.Invoice
		consumerTotalsStorage *mockConsumerTotalsStorage
	}
	tests := []struct {
		name          string
		fields        fields
		invoice       crypto.Invoice
		wantToPromise uint64
		wantDiff      uint64
		wantErr       bool
	}{
		{
			name: "bubbles totals storage errors",
			fields: fields{
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					err: errors.New("explosions everywhere"),
				},
			},
			invoice: crypto.Invoice{},
			wantErr: true,
		},
		{
			name: "assumes zero",
			fields: fields{
				consumerTotalsStorage: &mockConsumerTotalsStorage{},
			},
			invoice:       crypto.Invoice{AgreementTotal: 10},
			wantErr:       false,
			wantDiff:      10,
			wantToPromise: 10,
		},
		{
			name: "calculates correctly with different grand total",
			fields: fields{
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					res: 100,
				},
			},
			invoice:       crypto.Invoice{AgreementTotal: 10},
			wantErr:       false,
			wantDiff:      10,
			wantToPromise: 110,
		},
		{
			name: "calculates correctly with previous invoice",
			fields: fields{
				lastInvoice: crypto.Invoice{AgreementID: 111, AgreementTotal: 111},
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					res: 100,
				},
			},
			invoice:       crypto.Invoice{AgreementID: 111, AgreementTotal: 120},
			wantErr:       false,
			wantDiff:      9,
			wantToPromise: 109,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emt := &InvoicePayer{
				deps: InvoicePayerDeps{
					ConsumerTotalsStorage: tt.fields.consumerTotalsStorage,
					Peer:                  tt.fields.peer,
				},
			}
			emt.lastInvoice = tt.fields.lastInvoice
			gotToPromise, gotDiff, err := emt.calculateAmountToPromise(tt.invoice)
			if (err != nil) != tt.wantErr {
				t.Errorf("InvoicePayer.calculateAmountToPromise() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotToPromise != tt.wantToPromise {
				t.Errorf("InvoicePayer.calculateAmountToPromise() gotToPromise = %v, want %v", gotToPromise, tt.wantToPromise)
			}
			if gotDiff != tt.wantDiff {
				t.Errorf("InvoicePayer.calculateAmountToPromise() gotDiff = %v, want %v", gotDiff, tt.wantDiff)
			}
		})
	}
}

func TestInvoicePayer_issueExchangeMessage_publishesEvents(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestInvoicePayer_issueExchangeMessage_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	err = ks.Unlock(acc, "")
	assert.Nil(t, err)

	peerID := identity.FromAddress("0x01")
	sessionID := "100"

	mp := &mockPublisher{
		publicationChan: make(chan testEvent, 10),
	}
	emt := &InvoicePayer{
		deps: InvoicePayerDeps{
			PeerExchangeMessageSender: &MockPeerExchangeMessageSender{
				chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
			},
			ConsumerTotalsStorage: &mockConsumerTotalsStorage{
				bus: mp,
			},
			Ks:        ks,
			EventBus:  mp,
			Identity:  identity.FromAddress(acc.Address.Hex()),
			Peer:      peerID,
			SessionID: sessionID,
		},
	}
	emt.lastInvoice = crypto.Invoice{
		AgreementTotal: 10,
	}
	err = emt.issueExchangeMessage(crypto.Invoice{
		AgreementTotal: 15,
		Hashlock:       "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C",
	})
	assert.NoError(t, err)

	ev := <-mp.publicationChan
	assert.Equal(t, event.AppTopicInvoicePaid, ev.name)
	assert.EqualValues(t, event.AppEventInvoicePaid{
		ConsumerID: emt.deps.Identity,
		SessionID:  sessionID,
		Invoice: crypto.Invoice{
			AgreementTotal: 15,
			Hashlock:       "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C",
		},
	}, ev.value)

	ev = <-mp.publicationChan
	assert.Equal(t, event.AppTopicGrandTotalChanged, ev.name)
	assert.EqualValues(t, event.AppEventGrandTotalChanged{
		ConsumerID: emt.deps.Identity,
		Current:    5,
	}, ev.value)
}

func TestInvoicePayer_issueExchangeMessage(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestInvoicePayer_issueExchangeMessage_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	err = ks.Unlock(acc, "")
	assert.Nil(t, err)

	peerID := identity.FromAddress("0x01")

	type fields struct {
		peerExchangeMessageSender *MockPeerExchangeMessageSender
		keystore                  *identity.Keystore
		identity                  identity.Identity
		peer                      identity.Identity
		lastInvoice               crypto.Invoice
		consumerTotalsStorage     *mockConsumerTotalsStorage
	}
	type args struct {
		invoice crypto.Invoice
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		wantMsg *crypto.ExchangeMessage
	}{
		{
			name: "bubbles exchange message creation errors",
			fields: fields{
				identity: identity.FromAddress(""),
				peer:     peerID,
				keystore: ks,
				peerExchangeMessageSender: &MockPeerExchangeMessageSender{
					chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
				},
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					bus: eventbus.New(),
				},
			},
			wantErr: true,
		},
		{
			name: "ignores sending errors",
			fields: fields{
				identity: identity.FromAddress(acc.Address.Hex()),
				peer:     peerID,
				keystore: ks,
				peerExchangeMessageSender: &MockPeerExchangeMessageSender{
					chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
					mockError:     errors.New("explosions everywhere"),
				},
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					bus: eventbus.New(),
				},
			},
			wantErr: false,
		},
		{
			name: "sends exchange message",
			fields: fields{
				identity: identity.FromAddress(acc.Address.Hex()),
				peer:     peerID,
				keystore: ks,
				peerExchangeMessageSender: &MockPeerExchangeMessageSender{
					chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
				},
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					bus: eventbus.New(),
				},
			},
			args: args{
				invoice: crypto.Invoice{
					AgreementTotal: 15,
					Hashlock:       "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emt := &InvoicePayer{
				deps: InvoicePayerDeps{
					PeerExchangeMessageSender: tt.fields.peerExchangeMessageSender,
					ConsumerTotalsStorage:     tt.fields.consumerTotalsStorage,
					Peer:                      tt.fields.peer,
					Ks:                        tt.fields.keystore,
					Identity:                  tt.fields.identity,
					EventBus:                  mocks.NewEventBus(),
				},
			}
			emt.lastInvoice = tt.fields.lastInvoice
			if err := emt.issueExchangeMessage(tt.args.invoice); (err != nil) != tt.wantErr {
				t.Errorf("InvoicePayer.issueExchangeMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantMsg != nil {
				errMsg := "InvoicePayer.issueExchangeMessage() error"
				msg := <-tt.fields.peerExchangeMessageSender.chanToWriteTo
				assert.True(t, len(msg.Signature) > 0, errMsg)
				assert.True(t, len(msg.Promise.Signature) > 0, errMsg)
				assert.Equal(t, tt.fields.peer, msg.Provider, errMsg)
				assert.Equal(t, tt.args.invoice.AgreementTotal, msg.AgreementTotal, errMsg)
				assert.Equal(t, tt.args.invoice.AgreementTotal, msg.Promise.Amount, errMsg)
				assert.Equal(t, tt.args.invoice.Hashlock, msg.Promise.Hashlock, errMsg)
			}
		})
	}
}

type mockConsumerTotalsStorage struct {
	res     uint64
	resLock sync.Mutex
	bus     eventbus.Publisher

	err        error
	calledWith uint64
}

func (mcts *mockConsumerTotalsStorage) Store(id identity.Identity, accountantID common.Address, amount uint64) error {
	mcts.calledWith = amount
	go mcts.bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		Current:      amount,
		AccountantID: accountantID,
		ConsumerID:   id,
	})
	return nil
}

func (mcts *mockConsumerTotalsStorage) Get(id identity.Identity, accountantID common.Address) (uint64, error) {
	mcts.resLock.Lock()
	defer mcts.resLock.Unlock()
	return mcts.res, mcts.err
}

func (mcts *mockConsumerTotalsStorage) setResult(in uint64) {
	mcts.resLock.Lock()
	defer mcts.resLock.Unlock()
	mcts.res = in
}

type mockTimeTracker struct {
	timeToReturn time.Duration
}

func (mtt *mockTimeTracker) StartTracking() {

}
func (mtt *mockTimeTracker) Elapsed() time.Duration {
	return mtt.timeToReturn
}

func Test_estimateInvoiceTolerance(t *testing.T) {
	type args struct {
		elapsed     time.Duration
		transferred DataTransferred
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{"Zero time, zero data",
			args{
				0 * time.Second,
				DataTransferred{0, 0}},
			3},

		{"1 sec, 0 bytes",
			args{
				1 * time.Second,
				DataTransferred{0, 0}},
			1.6109756097560976},

		{"1 sec, 2 000 bytes",
			args{
				1 * time.Second,
				DataTransferred{1000, 1000}},
			1.6100149009391526},

		{"1 sec, 2 000 000 bytes",
			args{
				1 * time.Second,
				DataTransferred{1000000, 1000000}},
			1.6246823767314633},

		{"1 sec, 20 000 000 bytes",
			args{
				1 * time.Second,
				DataTransferred{10000000, 10000000}},
			1.7396867763477881},

		{"1 sec, 200 000 000 bytes",
			args{
				1 * time.Second,
				DataTransferred{100000000, 100000000}},
			2.2084123020547852},

		{"2 min, 0 bytes",
			args{
				2 * time.Minute,
				DataTransferred{0, 0}},
			1.4443089430894309},

		{"2 min, 2 000 bytes",
			args{
				2 * time.Minute,
				DataTransferred{1000, 1000}},
			1.4433334575096612},

		{"2 min, 2 000 000 bytes",
			args{
				2 * time.Minute,
				DataTransferred{1000000, 1000000}},
			1.4434574942587659},

		{"2 min, 20 000 000 bytes",
			args{
				2 * time.Minute,
				DataTransferred{10000000, 10000000}},
			1.4445735567021262},

		{"2 min, 200 000 000 bytes",
			args{
				2 * time.Minute,
				DataTransferred{100000000, 100000000}},
			1.455598661303886},

		{"20 min, 0 bytes",
			args{
				20 * time.Minute,
				DataTransferred{0, 0}},
			1.1585946573751453},

		{"20 min, 2 000 bytes",
			args{
				20 * time.Minute,
				DataTransferred{1000, 1000}},
			1.1576190600366817},

		{"20 min, 2 000 000 bytes",
			args{
				20 * time.Minute,
				DataTransferred{1000000, 1000000}},
			1.1576314650991801},

		{"20 min, 20 000 000 bytes",
			args{
				20 * time.Minute,
				DataTransferred{10000000, 10000000}},
			1.15774320854448},

		{"20 min, 200 000 000 bytes",
			args{
				20 * time.Minute,
				DataTransferred{100000000, 100000000}},
			1.1588592709878404},

		{"200 min, 200 000 000 bytes",
			args{
				200 * time.Minute,
				DataTransferred{100000000, 100000000}},
			1.115099285303542},

		{"1 min, 200 000 000 bytes",
			args{
				1 * time.Minute,
				DataTransferred{50000000, 50000000}},
			1.6222653279705525},

		{"1 min, 2 000 000 000 bytes",
			args{
				1 * time.Minute,
				DataTransferred{100000000, 100000000}},
			1.6342334250351986},

		{"1 min, 20 000 000 000 bytes",
			args{
				1 * time.Minute,
				DataTransferred{1000000000, 1000000000}},
			1.8089443281831476},

		{"10 min, 20 000 000 000 bytes",
			args{
				10 * time.Minute,
				DataTransferred{1000000000, 1000000000}},
			1.2251425159442896},

		{"6 hours, 20 000 000 000 bytes",
			args{
				6 * time.Hour,
				DataTransferred{1000000000, 1000000000}},
			1.1134594760857283},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := estimateInvoiceTolerance(tt.args.elapsed, tt.args.transferred); got != tt.want {
				t.Errorf("estimateInvoiceTolerance() = %v, want %v", got, tt.want)
			}
		})
	}
}
