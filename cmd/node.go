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

package cmd

import (
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/tequilapi"
)

// NATPinger allows to send nat pings as well as stop it
type NATPinger interface {
	Stop()
}

// Publisher is responsible for publishing given events
type Publisher interface {
	Publish(topic string, data interface{})
}

// SleepNotifier notifies node about pending sleep events
type SleepNotifier interface {
	Start()
	Stop()
}

// NewNode function creates new Mysterium node by given options
func NewNode(connectionManager connection.MultiManager, connectionDiagManager connection.DiagManager, tequilapiServer tequilapi.APIServer, publisher Publisher, uiServer UIServer, notifier SleepNotifier) *Node {
	return &Node{
		connectionManager:     connectionManager,
		connectionDiagManager: connectionDiagManager,

		httpAPIServer: tequilapiServer,
		publisher:     publisher,
		uiServer:      uiServer,
		sleepNotifier: notifier,
	}
}

// Node represent entrypoint for Mysterium node with top level components
type Node struct {
	connectionManager     connection.MultiManager
	connectionDiagManager connection.DiagManager

	httpAPIServer tequilapi.APIServer
	publisher     Publisher
	uiServer      UIServer
	sleepNotifier SleepNotifier
}

// Start starts Mysterium node (Tequilapi service, fetches location)
func (node *Node) Start() error {
	go node.sleepNotifier.Start()
	node.httpAPIServer.StartServing()

	node.uiServer.Serve()
	node.publisher.Publish(event.AppTopicNode, event.Payload{Status: event.StatusStarted})

	return nil
}

// Wait blocks until Mysterium node is stopped
func (node *Node) Wait() error {
	defer node.publisher.Publish(event.AppTopicNode, event.Payload{Status: event.StatusStopped})
	return node.httpAPIServer.Wait()
}

// Kill stops Mysterium node
func (node *Node) Kill() error {
	err := node.connectionManager.Disconnect(-1)
	if err != nil {
		switch err {
		case connection.ErrNoConnection:
			log.Info().Msg("No active connection - proceeding")
		default:
			return err
		}
	} else {
		log.Info().Msg("Connection closed")
	}

	node.httpAPIServer.Stop()
	log.Info().Msg("API stopped")

	node.uiServer.Stop()
	log.Info().Msg("Web UI server stopped")

	node.sleepNotifier.Stop()
	log.Info().Msg("Sleep notifier stopped")

	return nil
}
