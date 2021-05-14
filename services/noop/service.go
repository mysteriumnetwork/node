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
	"encoding/json"
	"net"
	"sync"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ErrAlreadyStarted is the error we return when the start is called multiple times
var ErrAlreadyStarted = errors.New("service already started")

// NewManager creates new instance of Noop service
func NewManager() *Manager {
	return &Manager{}
}

// Manager represents entrypoint for Noop service
type Manager struct {
	process sync.WaitGroup
}

// ProvideConfig provides the session configuration
func (manager *Manager) ProvideConfig(_ string, _ json.RawMessage, _ *net.UDPConn) (*service.ConfigParams, error) {
	return &service.ConfigParams{}, nil
}

// Serve starts service - does block
func (manager *Manager) Serve(instance *service.Instance) error {
	manager.process.Add(1)
	log.Info().Msg("Noop service started successfully")
	manager.process.Wait()
	return nil
}

// Stop stops service
func (manager *Manager) Stop() error {
	manager.process.Done()
	log.Info().Msg("Noop service stopped")
	return nil
}
