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

package connectivity

import (
	"github.com/mysteriumnetwork/node/communication"
)

// StatusSender is responsible for sending session connectivity status to other peer.
type StatusSender interface {
	Send(dialog communication.Dialog, msg *StatusMessage)
}

// NewStatusSender creates StatusSender instance.
func NewStatusSender() StatusSender {
	return &statusSender{}
}

type statusSender struct{}

// Send sends status message to other peer via broker.
func (s *statusSender) Send(dialog communication.Dialog, msg *StatusMessage) {
	producer := &statusProducer{
		message: msg,
	}
	if err := dialog.Send(producer); err != nil {
		log.Errorf("could not send connectivity status: %v", err)
	}
}
