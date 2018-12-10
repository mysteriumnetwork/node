/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package session

import (
	"github.com/mysteriumnetwork/node/communication"
)

// ManagerFactory initiates session Manager instance during runtime
type ManagerFactory func(dialog communication.Dialog) *Manager

// NewDialogHandler constructs handler which gets all incoming dialogs and starts handling them
func NewDialogHandler(sessionManagerFactory ManagerFactory, configConsumer ConfigConsumer) *handler {
	return &handler{
		sessionManagerFactory: sessionManagerFactory,
		configConsumer:        configConsumer,
	}
}

type handler struct {
	sessionManagerFactory ManagerFactory
	configConsumer        ConfigConsumer
}

// Handle starts serving services in given Dialog instance
func (handler *handler) Handle(dialog communication.Dialog) error {
	return handler.subscribeSessionRequests(dialog)
}

func (handler *handler) subscribeSessionRequests(dialog communication.Dialog) error {
	err := dialog.Respond(
		&createConsumer{
			sessionCreator: handler.sessionManagerFactory(dialog),
			peerID:         dialog.PeerID(),
			configConsumer: handler.configConsumer,
		},
	)

	if err != nil {
		return err
	}

	return dialog.Respond(
		&destroyConsumer{
			SessionDestroyer: handler.sessionManagerFactory(dialog),
			PeerID:           dialog.PeerID(),
		},
	)
}
