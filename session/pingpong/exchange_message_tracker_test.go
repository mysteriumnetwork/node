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
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/pingpong/paydef"
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

func Test_ExchangeMessageTracker_Start_Stop(t *testing.T) {
	dir, err := ioutil.TempDir("", "exchange_message_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	mockSender := &MockPeerExchangeMessageSender{
		chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
	}

	invoiceChan := make(chan crypto.Invoice)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(time.Now)
	invoiceStorage := NewConsumerInvoiceStorage(NewInvoiceStorage(bolt))
	totalsStorage := NewConsumerTotalsStorage(bolt)
	deps := ExchangeMessageTrackerDeps{
		InvoiceChan:               invoiceChan,
		PeerExchangeMessageSender: mockSender,
		ConsumerInvoiceStorage:    invoiceStorage,
		ConsumerTotalsStorage:     totalsStorage,
		TimeTracker:               &tracker,
		Ks:                        ks,
		Identity:                  identity.FromAddress(acc.Address.Hex()),
		ChannelImplementation:     acc.Address.Hex(),
		RegistryAddress:           acc.Address.Hex(),
		Peer:                      identity.FromAddress("0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"),
		PaymentInfo:               paydef.PaymentRate{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
	}
	exchangeMessageTracker := NewExchangeMessageTracker(deps)

	go func() {
		time.Sleep(time.Nanosecond * 10)
		exchangeMessageTracker.Stop()
	}()

	err = exchangeMessageTracker.Start()
	assert.Nil(t, err)
}

func Test_ExchangeMessageTracker_SendsMessage(t *testing.T) {
	dir, err := ioutil.TempDir("", "exchange_message_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
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

	tracker := session.NewTracker(time.Now)
	invoiceStorage := NewConsumerInvoiceStorage(NewInvoiceStorage(bolt))
	totalsStorage := NewConsumerTotalsStorage(bolt)
	deps := ExchangeMessageTrackerDeps{
		InvoiceChan:               invoiceChan,
		PeerExchangeMessageSender: mockSender,
		ConsumerInvoiceStorage:    invoiceStorage,
		ConsumerTotalsStorage:     totalsStorage,
		TimeTracker:               &tracker,
		Ks:                        ks,
		ChannelImplementation:     acc.Address.Hex(),
		RegistryAddress:           acc.Address.Hex(),
		Identity:                  identity.FromAddress(acc.Address.Hex()),
		Peer:                      identity.FromAddress("0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"),
		PaymentInfo:               paydef.PaymentRate{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
	}
	exchangeMessageTracker := NewExchangeMessageTracker(deps)

	mockInvoice := crypto.Invoice{
		AgreementID:    1,
		AgreementTotal: 0,
		TransactorFee:  0,
		Hashlock:       "0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C",
		Provider:       deps.Peer.Address,
	}

	testDone := make(chan struct{}, 0)

	defer exchangeMessageTracker.Stop()
	go func() {
		err := exchangeMessageTracker.Start()
		assert.Nil(t, err)
		testDone <- struct{}{}
	}()

	invoiceChan <- mockInvoice

	exchangeMessage := <-mockSender.chanToWriteTo
	exchangeMessageTracker.Stop()
	addr, err := exchangeMessage.RecoverConsumerIdentity()
	assert.Nil(t, err)

	assert.Equal(t, acc.Address.Hex(), addr.Hex())

	<-testDone
}

func Test_ExchangeMessageTracker_BubblesErrors(t *testing.T) {
	dir, err := ioutil.TempDir("", "exchange_message_tracker_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	mockSender := &MockPeerExchangeMessageSender{
		chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
	}

	invoiceChan := make(chan crypto.Invoice)
	bolt, err := boltdb.NewStorage(dir)
	assert.Nil(t, err)
	defer bolt.Close()

	tracker := session.NewTracker(time.Now)
	invoiceStorage := NewConsumerInvoiceStorage(NewInvoiceStorage(bolt))
	totalsStorage := NewConsumerTotalsStorage(bolt)
	deps := ExchangeMessageTrackerDeps{
		InvoiceChan:               invoiceChan,
		PeerExchangeMessageSender: mockSender,
		ConsumerInvoiceStorage:    invoiceStorage,
		ConsumerTotalsStorage:     totalsStorage,
		TimeTracker:               &tracker,
		Ks:                        ks,
		ChannelImplementation:     acc.Address.Hex(),
		RegistryAddress:           acc.Address.Hex(),
		Identity:                  identity.FromAddress(acc.Address.Hex()),
		Peer:                      identity.FromAddress("0x441Da57A51e42DAB7Daf55909Af93A9b00eEF23C"),
		PaymentInfo:               paydef.PaymentRate{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
	}
	exchangeMessageTracker := NewExchangeMessageTracker(deps)
	defer exchangeMessageTracker.Stop()
	errChan := make(chan error)
	go func() { errChan <- exchangeMessageTracker.Start() }()

	invoiceChan <- crypto.Invoice{}

	err = <-errChan
	assert.Error(t, err)
}

func TestExchangeMessageTracker_isInvoiceOK(t *testing.T) {
	type fields struct {
		peer        identity.Identity
		timeTracker timeTracker
		paymentInfo paydef.PaymentRate
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
			name:   "errors on non zero fee",
			fields: fields{},
			invoice: crypto.Invoice{
				TransactorFee: 1,
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
				paymentInfo: paydef.PaymentRate{
					Duration: time.Minute,
					Price:    money.NewMoney(100000, money.CurrencyMyst),
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
				paymentInfo: paydef.PaymentRate{
					Duration: time.Minute,
					Price:    money.NewMoney(100000, money.CurrencyMyst),
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
			emt := &ExchangeMessageTracker{
				timeTracker: tt.fields.timeTracker,
				paymentInfo: tt.fields.paymentInfo,
				peer:        tt.fields.peer,
			}
			if err := emt.isInvoiceOK(tt.invoice); (err != nil) != tt.wantErr {
				t.Errorf("ExchangeMessageTracker.isInvoiceOK() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExchangeMessageTracker_getGrandTotalPromised(t *testing.T) {
	type fields struct {
		consumerTotalsStorage consumerTotalsStorage
	}
	tests := []struct {
		name    string
		fields  fields
		want    uint64
		wantErr bool
	}{
		{
			name: "returns the amount from storage",
			fields: fields{
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					res: 10,
				},
			},
			want:    10,
			wantErr: false,
		},
		{
			name: "returns the error from storage",
			fields: fields{
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					err: errors.New("some error"),
				},
			},
			wantErr: true,
		},
		{
			name: "returns zero if not found",
			fields: fields{
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					err: ErrNotFound,
				},
			},
			wantErr: false,
			want:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emt := &ExchangeMessageTracker{
				consumerTotalsStorage: tt.fields.consumerTotalsStorage,
			}
			got, err := emt.getGrandTotalPromised()
			if (err != nil) != tt.wantErr {
				t.Errorf("ExchangeMessageTracker.getGranpaydef.alPromised() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExchangeMessageTracker.getGranpaydef.alPromised() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExchangeMessageTracker_incrementGrandTotalPromised(t *testing.T) {
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
					err: errors.New("some error"),
				},
			},
			wantErr: true,
		},
		{
			name: "adds to zero if not found",
			fields: fields{
				consumerTotalsStorage: &mockConsumerTotalsStorage{
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
			emt := &ExchangeMessageTracker{
				consumerTotalsStorage: tt.fields.consumerTotalsStorage,
			}
			if err := emt.incrementGrandTotalPromised(tt.args.amount); (err != nil) != tt.wantErr {
				t.Errorf("ExchangeMessageTracker.incrementGranpaydef.alPromised() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := tt.fields.consumerTotalsStorage.calledWith
			if got != tt.want {
				t.Errorf("ExchangeMessageTracker.incrementGranpaydef.alPromised() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExchangeMessageTracker_calculateAmountToPromise(t *testing.T) {
	type fields struct {
		peer                   identity.Identity
		consumerInvoiceStorage *mockConsumerInvoiceStorage
		consumerTotalsStorage  *mockConsumerTotalsStorage
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
			name: "bubbles invoice storage errors",
			fields: fields{
				consumerInvoiceStorage: &mockConsumerInvoiceStorage{
					err: errors.New("explosions everywhere"),
				},
			},
			invoice: crypto.Invoice{},
			wantErr: true,
		},
		{
			name: "bubbles totals storage errors",
			fields: fields{
				consumerInvoiceStorage: &mockConsumerInvoiceStorage{},
				consumerTotalsStorage: &mockConsumerTotalsStorage{
					err: errors.New("explosions everywhere"),
				},
			},
			invoice: crypto.Invoice{},
			wantErr: true,
		},
		{
			name: "ignores bolt not found errors",
			fields: fields{
				consumerInvoiceStorage: &mockConsumerInvoiceStorage{
					err: ErrNotFound,
				},
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
				consumerInvoiceStorage: &mockConsumerInvoiceStorage{
					err: ErrNotFound,
				},
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
				consumerInvoiceStorage: &mockConsumerInvoiceStorage{
					res: crypto.Invoice{AgreementID: 111, AgreementTotal: 111},
				},
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
			emt := &ExchangeMessageTracker{
				peer:                   tt.fields.peer,
				consumerInvoiceStorage: tt.fields.consumerInvoiceStorage,
				consumerTotalsStorage:  tt.fields.consumerTotalsStorage,
			}
			gotToPromise, gotDiff, err := emt.calculateAmountToPromise(tt.invoice)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExchangeMessageTracker.calculateAmountToPromise() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotToPromise != tt.wantToPromise {
				t.Errorf("ExchangeMessageTracker.calculateAmountToPromise() gotToPromise = %v, want %v", gotToPromise, tt.wantToPromise)
			}
			if gotDiff != tt.wantDiff {
				t.Errorf("ExchangeMessageTracker.calculateAmountToPromise() gotDiff = %v, want %v", gotDiff, tt.wantDiff)
			}
		})
	}
}

func TestExchangeMessageTracker_issueExchangeMessage(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestExchangeMessageTracker_issueExchangeMessage_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	err = ks.Unlock(acc, "")
	assert.Nil(t, err)

	peerID := identity.FromAddress("0x01")

	type fields struct {
		peerExchangeMessageSender *MockPeerExchangeMessageSender
		keystore                  *keystore.KeyStore
		identity                  identity.Identity
		peer                      identity.Identity
		consumerInvoiceStorage    *mockConsumerInvoiceStorage
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
			name: "bubbles calculation errors",
			fields: fields{
				identity: identity.FromAddress(acc.Address.Hex()),
				peer:     peerID,
				keystore: ks,
				peerExchangeMessageSender: &MockPeerExchangeMessageSender{
					chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
				},
				consumerInvoiceStorage: &mockConsumerInvoiceStorage{
					err: errors.New("explosions everywhere"),
				},
			},
			wantErr: true,
		},
		{
			name: "bubbles exchange message creation errors",
			fields: fields{
				identity: identity.FromAddress(""),
				peer:     peerID,
				keystore: ks,
				peerExchangeMessageSender: &MockPeerExchangeMessageSender{
					chanToWriteTo: make(chan crypto.ExchangeMessage, 10),
				},
				consumerInvoiceStorage: &mockConsumerInvoiceStorage{},
				consumerTotalsStorage:  &mockConsumerTotalsStorage{},
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
				consumerInvoiceStorage: &mockConsumerInvoiceStorage{},
				consumerTotalsStorage:  &mockConsumerTotalsStorage{},
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
				consumerInvoiceStorage: &mockConsumerInvoiceStorage{},
				consumerTotalsStorage:  &mockConsumerTotalsStorage{},
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
			emt := &ExchangeMessageTracker{
				peerExchangeMessageSender: tt.fields.peerExchangeMessageSender,
				keystore:                  tt.fields.keystore,
				identity:                  tt.fields.identity,
				peer:                      tt.fields.peer,
				consumerInvoiceStorage:    tt.fields.consumerInvoiceStorage,
				consumerTotalsStorage:     tt.fields.consumerTotalsStorage,
			}
			if err := emt.issueExchangeMessage(tt.args.invoice); (err != nil) != tt.wantErr {
				t.Errorf("ExchangeMessageTracker.issueExchangeMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantMsg != nil {
				errMsg := "ExchangeMessageTracker.issueExchangeMessage() error"
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
	res        uint64
	err        error
	calledWith uint64
}

func (mcts *mockConsumerTotalsStorage) Store(providerAddress string, amount uint64) error {
	mcts.calledWith = amount
	return nil
}

func (mcts *mockConsumerTotalsStorage) Get(providerAddress string) (uint64, error) {
	return mcts.res, mcts.err
}

type mockConsumerInvoiceStorage struct {
	res        crypto.Invoice
	err        error
	calledWith crypto.Invoice
}

func (mcis *mockConsumerInvoiceStorage) Store(providerIdentity identity.Identity, invoice crypto.Invoice) error {
	mcis.calledWith = invoice
	return nil
}

func (mcis *mockConsumerInvoiceStorage) Get(providerAddress identity.Identity) (crypto.Invoice, error) {
	return mcis.res, mcis.err
}

type mockTimeTracker struct {
	timeToReturn time.Duration
}

func (mtt *mockTimeTracker) StartTracking() {

}
func (mtt *mockTimeTracker) Elapsed() time.Duration {
	return mtt.timeToReturn
}
