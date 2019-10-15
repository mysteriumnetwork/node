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
	"time"

	"github.com/mysteriumnetwork/node/communication"
)

// StatusSubscriber is responsible for handling status events for the peer.
type StatusSubscriber interface {
	Subscribe(dialog communication.Dialog)
}

// NewStatusSubscriber returns new StatusSubscriber instance.
func NewStatusSubscriber(statusStorage StatusStorage) StatusSubscriber {
	return &statusSubscriber{
		statusStorage: statusStorage,
	}
}

type statusSubscriber struct {
	statusStorage StatusStorage
}

func (s *statusSubscriber) Subscribe(dialog communication.Dialog) {
	consumer := &statusConsumer{
		callback: func(msg *StatusMessage) {
			entry := StatusEntry{
				PeerID:       dialog.PeerID(),
				SessionID:    msg.SessionID,
				StatusCode:   msg.StatusCode,
				Message:      msg.Message,
				CreatedAtUTC: time.Now().UTC(),
			}
			s.statusStorage.AddStatusEntry(entry)
		},
	}
	if err := dialog.Receive(consumer); err != nil {
		log.Errorf("could not receive connectivity status: %v", err)
		return
	}
}
