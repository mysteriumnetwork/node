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

package node

import (
	log "github.com/cihub/seelog"
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/tequilapi"
	tequilapi_endpoints "github.com/mysteriumnetwork/node/tequilapi/endpoints"
	"github.com/mysteriumnetwork/node/utils"
)

// NewNode function creates new Mysterium node by given options
func NewNode(
	connectionManager connection.Manager,
	tequilapiServer tequilapi.APIServer,
	router *httprouter.Router,
	originalLocationCache location.Cache,
) *Node {
	return &Node{
		router:                router,
		connectionManager:     connectionManager,
		httpAPIServer:         tequilapiServer,
		originalLocationCache: originalLocationCache,
	}
}

// Node represent entrypoint for Mysterium node with top level components
type Node struct {
	connectionManager     connection.Manager
	router                *httprouter.Router
	httpAPIServer         tequilapi.APIServer
	originalLocationCache location.Cache
}

// Start starts Mysterium node (Tequilapi service, fetches location)
func (node *Node) Start() error {
	originalLocation, err := node.originalLocationCache.RefreshAndGet()
	if err != nil {
		log.Warn("Failed to detect original country: ", err)
	} else {
		log.Info("Original country detected: ", originalLocation.Country)
	}

	tequilapi_endpoints.AddRouteForStop(node.router, utils.SoftKiller(node.Kill))

	err = node.httpAPIServer.StartServing()
	if err != nil {
		return err
	}

	address, err := node.httpAPIServer.Address()
	if err != nil {
		return err
	}

	log.Infof("Api started on: %v", address)

	return nil
}

// Wait blocks until Mysterium node is stopped
func (node *Node) Wait() error {
	return node.httpAPIServer.Wait()
}

// Kill stops Mysterium node
func (node *Node) Kill() error {
	err := node.connectionManager.Disconnect()
	if err != nil {
		switch err {
		case connection.ErrNoConnection:
			log.Info("No active connection - proceeding")
		default:
			return err
		}
	} else {
		log.Info("Connection closed")
	}

	node.httpAPIServer.Stop()
	log.Info("Api stopped")

	return nil
}
