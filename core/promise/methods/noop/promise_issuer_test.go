/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package noop

import (
	"errors"
	"testing"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/promise"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
)

var (
	providerID = identity.FromAddress("provider-id")
	proposal   = dto.ServiceProposal{
		ProviderID: providerID.Address,
	}
)

var _ connection.PromiseIssuer = &PromiseIssuer{}

func TestPromiseIssuer_Start_SubscriptionFails(t *testing.T) {
	dialog := &fakeDialog{
		onReceiveReturnError: errors.New("reject subscriptions"),
	}
	issuer := &PromiseIssuer{Dialog: dialog}

	logs := make([]string, 0)
	logger := logconfig.ReplaceLogger(logconfig.NewLoggerCapture(&logs))
	defer logconfig.ReplaceLogger(logger)

	err := issuer.Start(proposal)
	assert.EqualError(t, err, "reject subscriptions")
	assert.Len(t, logs, 0)
}

func TestPromiseIssuer_Start_SubscriptionOfBalances(t *testing.T) {
	dialog := &fakeDialog{}
	issuer := &PromiseIssuer{Dialog: dialog}

	logs := make([]string, 0)
	logger := logconfig.ReplaceLogger(logconfig.NewLoggerCapture(&logs))
	defer logconfig.ReplaceLogger(logger)

	err := issuer.Start(proposal)
	assert.NoError(t, err)

	dialog.onReceiveLastConsumer.Consume(
		promise.BalanceMessage{1, true, testToken(10)},
	)
	assert.Len(t, logs, 1)
	assert.Equal(t, "[promise-issuer] Promise balance notified: 1000000000TEST", logs[0])
}

func testToken(amount float64) money.Money {
	return money.NewMoney(amount, money.Currency("TEST"))
}

type fakeDialog struct {
	onReceiveLastConsumer communication.MessageConsumer
	onReceiveReturnError  error
}

func (fd *fakeDialog) PeerID() identity.Identity {
	return providerID
}

func (fd *fakeDialog) Close() error {
	return nil
}

func (fd *fakeDialog) Receive(consumer communication.MessageConsumer) error {
	if fd.onReceiveReturnError != nil {
		return fd.onReceiveReturnError
	}

	fd.onReceiveLastConsumer = consumer
	return nil
}
func (fd *fakeDialog) Respond(consumer communication.RequestConsumer) error {
	return nil
}

func (fd *fakeDialog) Send(producer communication.MessageProducer) error {
	return nil
}

func (fd *fakeDialog) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	return nil, nil
}
