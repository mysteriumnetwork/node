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
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/payments/crypto"
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
		Peer:                      identity.FromAddress("some peer"),
		PaymentInfo:               dto.PaymentPerTime{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
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
		Identity:                  identity.FromAddress(acc.Address.Hex()),
		Peer:                      identity.FromAddress("some peer"),
		PaymentInfo:               dto.PaymentPerTime{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
	}
	exchangeMessageTracker := NewExchangeMessageTracker(deps)

	mockInvoice := crypto.Invoice{
		AgreementID:    1,
		AgreementTotal: 0,
		Fee:            0,
		Hashlock:       "lock",
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
		Identity:                  identity.FromAddress(acc.Address.Hex()),
		Peer:                      identity.FromAddress("some peer"),
		PaymentInfo:               dto.PaymentPerTime{Price: money.NewMoney(10, money.CurrencyMyst), Duration: time.Minute},
	}
	exchangeMessageTracker := NewExchangeMessageTracker(deps)
	defer exchangeMessageTracker.Stop()
	errChan := make(chan error)
	go func() { errChan <- exchangeMessageTracker.Start() }()

	invoiceChan <- crypto.Invoice{}

	err = <-errChan
	assert.Error(t, err)
}
