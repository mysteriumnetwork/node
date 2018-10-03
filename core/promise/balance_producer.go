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

// BalanceMessageProducer sends balance notification to thought communication channel.
type BalanceMessageProducer struct {
	Message BalanceMessage
}

// GetMessageEndpoint returns endpoint there to send messages
func (producer *BalanceMessageProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return balanceEndpoint
}

// Produce creates message which will be serialized to endpoint
func (producer *BalanceMessageProducer) Produce() (messagePtr interface{}) {
	return producer.Message
}
