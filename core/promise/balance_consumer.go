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

package promise

import (
	"github.com/mysteriumnetwork/node/communication"
)

// BalanceMessageConsumer process balance notification from communication channel.
type BalanceMessageConsumer struct {
}

// GetMessageEndpoint returns endpoint there to receive messages
func (consumer *BalanceMessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return balanceEndpoint
}

// NewMessage creates struct where message from endpoint will be serialized
func (consumer *BalanceMessageConsumer) NewMessage() (messagePtr interface{}) {
	var message BalanceMessage
	return &message
}

// Consume handles messages from endpoint
func (consumer *BalanceMessageConsumer) Consume(messagePtr interface{}) error {
	return nil
}
