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
	"github.com/mysteriumnetwork/node/identity"
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
	exchangeMessageTracker := NewExchangeMessageTracker(
		invoiceChan,
		mockSender,
		ks,
		identity.FromAddress(acc.Address.Hex()),
	)

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
	exchangeMessageTracker := NewExchangeMessageTracker(
		invoiceChan,
		mockSender,
		ks,
		identity.FromAddress(acc.Address.Hex()),
	)

	mockInvoice := crypto.Invoice{
		AgreementID:    1,
		AgreementTotal: 1,
		Fee:            1,
		Hashlock:       "lock",
		Provider:       "provider",
	}

	defer exchangeMessageTracker.Stop()
	go exchangeMessageTracker.Start()

	invoiceChan <- mockInvoice

	exchangeMessage := <-mockSender.chanToWriteTo
	addr, err := exchangeMessage.RecoverConsumerIdentity()
	assert.Nil(t, err)

	assert.Equal(t, acc.Address.Hex(), addr.Hex())
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
	exchangeMessageTracker := NewExchangeMessageTracker(
		invoiceChan,
		mockSender,
		ks,
		identity.FromAddress(acc.Address.Hex()),
	)

	defer exchangeMessageTracker.Stop()
	errChan := make(chan error)
	go func() { errChan <- exchangeMessageTracker.Start() }()

	invoiceChan <- crypto.Invoice{}

	err = <-errChan
	assert.Error(t, err)
}
