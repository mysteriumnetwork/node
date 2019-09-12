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
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
)

// PeerInvoiceSender allows to send invoices
type PeerInvoiceSender interface {
	Send(crypto.Invoice) error
}

// ErrExchangeWaitTimeout indicates that we did not get an exchange message in time
var ErrExchangeWaitTimeout = errors.New("did not get a new exchange message")

// ErrExchangeValidationFailed indicates that there was an error with the exchange signature
var ErrExchangeValidationFailed = errors.New("exchange validation failed")

// errBoltNotFound indicates that bolt did not find a record
var errBoltNotFound = errors.New("not found")

const chargePeriodLeeway = time.Hour * 2

// InvoiceTracker keeps tab of invoices and sends them to the consumer
type InvoiceTracker struct {
	peer                            identity.Identity
	stop                            chan struct{}
	peerInvoiceSender               PeerInvoiceSender
	exchangeMessageChan             chan crypto.ExchangeMessage
	chargePeriod                    time.Duration
	exchangeMessageWaitTimeout      time.Duration
	notReceivedExchangeMessageCount uint64
	maxNotReceivedExchangeMessages  uint64
	once                            sync.Once
}

// NewInvoiceTracker creates a new instancec of invoice tracker
func NewInvoiceTracker(
	peer identity.Identity,
	peerInvoiceSender PeerInvoiceSender,
	chargePeriod time.Duration,
	exchangeMessageChan chan crypto.ExchangeMessage,
	exchangeMessageWaitTimeout time.Duration) *InvoiceTracker {
	return &InvoiceTracker{
		peer:                           peer,
		stop:                           make(chan struct{}),
		peerInvoiceSender:              peerInvoiceSender,
		exchangeMessageChan:            exchangeMessageChan,
		exchangeMessageWaitTimeout:     exchangeMessageWaitTimeout,
		chargePeriod:                   chargePeriod,
		maxNotReceivedExchangeMessages: calculateMaxNotReceivedExchangeMessageCount(chargePeriodLeeway, chargePeriod),
	}
}

func calculateMaxNotReceivedExchangeMessageCount(chargeLeeway, chargePeriod time.Duration) uint64 {
	return uint64(math.Round(float64(chargeLeeway) / float64(chargePeriod)))
}

// Start stars the invoice tracker
func (it *InvoiceTracker) Start() error {
	log.Debug("Starting...")
	// give the consumer a second to start up his payments before sending the first request
	firstSend := time.After(time.Second)
	for {
		select {
		case <-firstSend:
			err := it.sendInvoiceExpectExchangeMessage()
			if err != nil {
				return err
			}
		case <-it.stop:
			return nil
		case <-time.After(it.chargePeriod):
			err := it.sendInvoiceExpectExchangeMessage()
			if err != nil {
				return err
			}
		}
	}
}

func (it *InvoiceTracker) markExchangeMessageNotReceived() {
	atomic.AddUint64(&it.notReceivedExchangeMessageCount, 1)
}

func (it *InvoiceTracker) resetNotReceivedExchangeMessageCount() {
	atomic.SwapUint64(&it.notReceivedExchangeMessageCount, 0)
}

func (it *InvoiceTracker) getNotReceivedExchangeMessageCount() uint64 {
	return atomic.LoadUint64(&it.notReceivedExchangeMessageCount)
}

func (it *InvoiceTracker) sendInvoiceExpectExchangeMessage() error {
	err := it.sendInvoice()
	if err != nil {
		return err
	}

	err = it.receiveExchangeMessageOrTimeout()
	if err != nil {
		handlerErr := it.handleExchangeMessageReceiveError(err)
		if handlerErr != nil {
			return err
		}
	} else {
		it.resetNotReceivedExchangeMessageCount()
	}
	return nil
}

func (it *InvoiceTracker) handleExchangeMessageReceiveError(err error) error {
	// if it's a timeout, we'll want to ignore it if we're not exceeding maxNotReceivedexchangeMessages
	if err == ErrExchangeWaitTimeout {
		it.markExchangeMessageNotReceived()
		if it.getNotReceivedExchangeMessageCount() >= it.maxNotReceivedExchangeMessages {
			return err
		}
		log.Warn("Failed to receive exchangeMessage: ", err)
		return nil
	}
	return err
}

func (it *InvoiceTracker) sendInvoice() error {
	// TODO: a ton of actions should go here

	// TODO: fill the fields
	return it.peerInvoiceSender.Send(crypto.Invoice{AgreementID: 1234})
}

func (it *InvoiceTracker) receiveExchangeMessageOrTimeout() error {
	select {
	case pm := <-it.exchangeMessageChan:
		if res := pm.ValidateExchangeMessage(common.HexToAddress(it.peer.Address)); !res {
			return ErrExchangeValidationFailed
		}
	case <-time.After(it.exchangeMessageWaitTimeout):
		return ErrExchangeWaitTimeout
	case <-it.stop:
		return nil
	}
	return nil
}

// Stop stops the invoice tracker
func (it *InvoiceTracker) Stop() {
	it.once.Do(func() {
		log.Debug("Stopping...")
		close(it.stop)
	})
}
