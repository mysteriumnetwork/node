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
	"sync"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
)

// PeerExchangeMessageSender allows for sending of exchange messages
type PeerExchangeMessageSender interface {
	Send(crypto.ExchangeMessage) error
}

// ExchangeMessageTracker keeps track of exchange messages and sends them to the provider
type ExchangeMessageTracker struct {
	stop                      chan struct{}
	invoiceChan               chan crypto.Invoice
	peerExchangeMessageSender PeerExchangeMessageSender
	once                      sync.Once
	keystore                  *keystore.KeyStore
	identity                  identity.Identity
}

// NewExchangeMessageTracker returns a new instance of exchange message tracker
func NewExchangeMessageTracker(invoiceChan chan crypto.Invoice, peerExchangeMessageSender PeerExchangeMessageSender, ks *keystore.KeyStore, identity identity.Identity) *ExchangeMessageTracker {
	return &ExchangeMessageTracker{
		stop:                      make(chan struct{}),
		peerExchangeMessageSender: peerExchangeMessageSender,
		invoiceChan:               invoiceChan,
		keystore:                  ks,
		identity:                  identity,
	}
}

// ErrInvoiceMissmatch represents an error that occurs when invoices do not match
var ErrInvoiceMissmatch = errors.New("invoice missmatch")

// Start starts the message exchange tracker. Blocks.
func (emt *ExchangeMessageTracker) Start() error {
	for {
		select {
		case <-emt.stop:
			return nil
		case balance := <-emt.invoiceChan:
			err := emt.issueExchangeMessage(balance)
			if err != nil {
				return err
			}
		}
	}
}

func (emt *ExchangeMessageTracker) issueExchangeMessage(invoice crypto.Invoice) error {
	msg, err := crypto.CreateExchangeMessage(invoice, 10, "", emt.keystore, common.HexToAddress(emt.identity.Address))
	if err != nil {
		return errors.Wrap(err, "could not create exchange message")
	}
	err = emt.peerExchangeMessageSender.Send(*msg)
	if err != nil {
		log.Warn("Failed to send exchange message: ", err)
	}

	return nil
}

// Stop stops the message tracker
func (emt *ExchangeMessageTracker) Stop() {
	emt.once.Do(func() {
		close(emt.stop)
	})
}
