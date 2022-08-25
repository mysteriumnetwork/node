//go:build !android

/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package router

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/router/network"
)

type manager struct {
	mu   sync.Mutex
	once sync.Once

	rules     []rule
	currentGW net.IP

	routingTable router

	gwCheckInterval time.Duration

	onceStop sync.Once
	stop     chan struct{}
}

type router interface {
	DiscoverGateway() (net.IP, error)
	ExcludeRule(ip, gw net.IP) error
	DeleteRule(ip, gw net.IP) error
}

type rule struct {
	ip    net.IP
	usage int
}

// NewManager creates a new instance of service that maintain routing table to match current state.
func NewManager() *manager {
	var r router = &network.RoutingTable{}

	if config.GetBool(config.FlagUserMode) || config.GetBool(config.FlagUserspace) {
		r = &network.RoutingTableRemote{}
	}

	return &manager{
		stop: make(chan struct{}),

		gwCheckInterval: 5 * time.Second,
		routingTable:    r,
	}
}

func (m *manager) ExcludeIP(ip net.IP) error {
	m.ensureStarted()
	m.mu.Lock()
	defer m.mu.Unlock()

	new := true

	for i, rule := range m.rules {
		if !rule.ip.Equal(ip) {
			continue
		}

		new = false
		m.rules[i].usage++

		break
	}

	if !new {
		return nil
	}

	if err := m.routingTable.ExcludeRule(ip, m.currentGW); err != nil {
		return fmt.Errorf("failed to exclude rule: %w", err)
	}

	m.rules = append(m.rules, rule{
		ip:    ip,
		usage: 1,
	})

	return nil
}

func (m *manager) RemoveExcludedIP(ip net.IP) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, rule := range m.rules {
		if !rule.ip.Equal(ip) {
			continue
		}

		m.rules[i].usage--

		if m.rules[i].usage == 0 {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)

			if err := m.routingTable.DeleteRule(ip, m.currentGW); err != nil {
				return fmt.Errorf("failed to remove excluded rule: %w", err)
			}
		}

		break
	}

	return nil
}

func (m *manager) ensureStarted() {
	m.once.Do(func() {
		m.forceCheckGW()

		go m.start()
	})
}

func (m *manager) start() {
	for {
		select {
		case <-time.After(m.gwCheckInterval):
			m.checkGW()
		case <-m.stop:
			return
		}
	}
}

func (m *manager) Stop() {
	if err := m.Clean(); err != nil {
		log.Error().Err(err).Msg("Failed to clean routing rules")
	}

	m.onceStop.Do(func() {
		close(m.stop)
	})
}

func (m *manager) Clean() (lastErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.clean(); err != nil {
		return fmt.Errorf("failed to clean routes: %w", err)
	}

	m.rules = nil

	return nil
}

func (m *manager) clean() (lastErr error) {
	for _, rule := range m.rules {
		err := m.routingTable.DeleteRule(rule.ip, m.currentGW)
		if err != nil {
			lastErr = err
			log.Error().Err(err).Msgf("Failed to delete route: %+v", rule)
		}
	}

	return lastErr
}

func (m *manager) apply(gw net.IP) (lastErr error) {
	for _, rule := range m.rules {
		err := m.routingTable.ExcludeRule(rule.ip, gw)
		if err != nil {
			lastErr = err
			log.Error().Err(err).Msgf("Failed to delete route: %+v", rule)
		}
	}

	m.currentGW = gw

	return lastErr
}

func (m *manager) forceCheckGW() {
	var currentGW net.IP

	for currentGW == nil {
		m.checkGW()

		m.mu.Lock()
		currentGW = m.currentGW
		m.mu.Unlock()
	}
}

func (m *manager) checkGW() {
	gw, err := m.routingTable.DiscoverGateway()
	if err != nil {
		log.Error().Err(err).Msg("Failed to detect system default gateway, keeping old value")
		return
	}

	if !m.currentGW.Equal(gw) && !gw.Equal(net.IPv4zero) {
		m.mu.Lock()
		defer m.mu.Unlock()

		log.Info().Msgf("Default gateway changed to %s, reconfiguring routes.", gw)

		if err := m.clean(); err != nil {
			log.Error().Err(err).Msg("Failed to clean routing rules")
		}

		if err := m.apply(gw); err != nil {
			log.Error().Err(err).Msg("Failed to apply new routing rules")
		}
	}
}
