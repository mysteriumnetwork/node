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

package mapping

import (
	"time"

	portmap "github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/rs/zerolog/log"
)

// StageName is used to indicate port mapping NAT traversal stage
const StageName = "port_mapping"

// DefaultConfig returns default port mapping config.
func DefaultConfig() *Config {
	return &Config{
		MapInterface:      portmap.Any(),
		MapLifetime:       20 * time.Minute,
		MapUpdateInterval: 15 * time.Minute,
	}
}

// Config represents port mapping config.
type Config struct {
	MapInterface      portmap.Interface
	MapLifetime       time.Duration
	MapUpdateInterval time.Duration
}

// PortMapper tries to map port using router's uPnP or NAT-PMP depending on given config map interface.
type PortMapper interface {
	// Map maps port for given protocol. It returns release func which
	// must be called when port no longer needed and ok which is true if
	// port mapping was successful.
	Map(protocol string, port int, name string) (release func(), ok bool)
}

// NewPortMapper returns port mapper instance.
func NewPortMapper(config *Config, publisher eventbus.Publisher) PortMapper {
	return &portMapper{
		config:    config,
		publisher: publisher,
	}
}

type portMapper struct {
	config    *Config
	publisher eventbus.Publisher
}

func (p *portMapper) Map(protocol string, port int, name string) (release func(), ok bool) {
	// Try add mapping first to determine if it is supported and
	// if permanent lease only is supported.
	permanent, err := p.addMapping(protocol, port, port, name)
	p.notify(err)
	if err != nil {
		return nil, false
	}

	// If only permanent lease is supported we don't need to update it in intervals.
	if permanent {
		return func() { p.deleteMapping(protocol, port, port) }, true
	}

	stopUpdate := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopUpdate:
				return
			case <-time.After(p.config.MapUpdateInterval):
				_, err := p.addMapping(protocol, port, port, name)
				p.notify(err)
			}
		}
	}()

	return func() {
		p.deleteMapping(protocol, port, port)
		close(stopUpdate)
	}, true
}

func (p *portMapper) notify(err error) {
	if err != nil {
		p.publisher.Publish(event.AppTopicTraversal, event.BuildFailureEvent(StageName, err))
	} else {
		p.publisher.Publish(event.AppTopicTraversal, event.BuildSuccessfulEvent(StageName))
	}
}

func (p *portMapper) addMapping(protocol string, extPort, intPort int, name string) (permanent bool, err error) {
	if err := p.config.MapInterface.AddMapping(protocol, extPort, intPort, name, p.config.MapLifetime); err != nil {
		log.Warn().Err(err).Msgf("Couldn't add port mapping for port %d: retrying with permanent lease", extPort)
		if err := p.config.MapInterface.AddMapping(protocol, extPort, intPort, name, 0); err != nil {
			// some gateways support only permanent leases
			log.Warn().Err(err).Msgf("Couldn't add port mapping for port %d", extPort)
			return false, err
		}
		return true, nil
	}
	log.Info().Msgf("Mapped network port: %d", extPort)
	return false, nil
}

func (p *portMapper) deleteMapping(protocol string, extPort, intPort int) {
	log.Debug().Msgf("Deleting port mapping for port: %d", extPort)
	if err := p.config.MapInterface.DeleteMapping(protocol, extPort, intPort); err != nil {
		log.Warn().Err(err).Msg("Couldn't delete port mapping")
	}
}
