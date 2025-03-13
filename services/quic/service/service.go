/*
 * Copyright (C) 2025 The "MysteriumNetwork/node" Authors.
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

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/services/quic/streams"
)

// NewManager creates new instance of Quic service
func NewManager(
	country string,
	eventBus eventbus.EventBus,
) *Manager {
	return &Manager{
		done:     make(chan struct{}),
		eventBus: eventBus,

		country:        country,
		sessionCleanup: map[string]func(){},
	}
}

// Manager represents an instance of Quic service
type Manager struct {
	done        chan struct{}
	startStopMu sync.Mutex

	eventBus eventbus.EventBus

	serviceInstance  *service.Instance
	sessionCleanup   map[string]func()
	sessionCleanupMu sync.Mutex

	country string
}

// ConsumerConfig represents configuration for consumer.
type ConsumerConfig struct {
	URL string `json:"url"`
}

// ProvideConfig provides the config for consumer and handles new Quic connection.
func (m *Manager) ProvideConfig(sessionID string, sessionConfig json.RawMessage, remoteConn p2p.ServiceConn) (*service.ConfigParams, error) {
	consumerConfig := ConsumerConfig{}
	err := json.Unmarshal(sessionConfig, &consumerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal wg consumer config")
	}

	cs := &connectServer{
		connectResponse: []byte("HTTP/1.1 200 OK\r\n\r\n"),
	}

	s := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           cs,
	}

	listener := &listener{
		ctx: context.TODO(),
		c:   remoteConn.(*streams.QuicConnection),
	}

	go func() {
		if err := s.Serve(listener); err != nil {
			log.Error().Err(err).Msg("Serve failed")
		}
	}()

	statsPublisher := newStatsPublisher(m.eventBus, time.Second)
	go statsPublisher.start(sessionID, cs)

	destroy := func() {
		log.Info().Msgf("Cleaning up quic session %s", sessionID)
		statsPublisher.stop()

		m.sessionCleanupMu.Lock()
		m.sessionCleanup[sessionID] = func() {}
		m.sessionCleanupMu.Unlock()
	}

	return &service.ConfigParams{SessionServiceConfig: nil, SessionDestroyCallback: destroy}, nil
}

// Serve starts service - does block
func (m *Manager) Serve(instance *service.Instance) error {
	log.Info().Msg("Quic: starting")

	m.startStopMu.Lock()
	m.serviceInstance = instance

	m.startStopMu.Unlock()
	log.Info().Msg("Quic: started")

	<-m.done

	return nil
}

// Stop stops service.
func (m *Manager) Stop() error {
	m.startStopMu.Lock()
	defer m.startStopMu.Unlock()

	cleanupWg := sync.WaitGroup{}

	// prevent concurrent iteration and write
	sessionCleanupCopy := make(map[string]func())
	if err := copier.Copy(&sessionCleanupCopy, m.sessionCleanup); err != nil {
		panic(err)
	}
	for k, v := range sessionCleanupCopy {
		cleanupWg.Add(1)
		go func(sessionID string, cleanup func()) {
			defer cleanupWg.Done()
			cleanup()
		}(k, v)
	}
	cleanupWg.Wait()

	close(m.done)
	return nil
}
