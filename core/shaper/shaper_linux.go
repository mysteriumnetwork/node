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

package shaper

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/go-wondershaper/wondershaper"
	"github.com/mysteriumnetwork/node/config"
)

type linuxShaper struct {
	ws          *wondershaper.Shaper
	listener    eventListener
	listenTopic string
}

type linuxShaperNoop struct{}

func (s *linuxShaperNoop) Start(interfaceName string) error {
	return nil
}

func (s *linuxShaperNoop) Clear(interfaceName string) {
	return
}

func create(listener eventListener) Shaper {
	// return a noop filter if userspace flag is set
	if config.GetBool(config.FlagUserspace) {
		return &linuxShaperNoop{}
	}

	ws := wondershaper.New()
	ws.Stdout = log.Logger
	ws.Stderr = log.Logger
	return &linuxShaper{
		ws:          ws,
		listener:    listener,
		listenTopic: config.AppTopicConfig(config.FlagShaperEnabled.Name),
	}
}

// Start applies shaping configuration on the specified interface and then continuously ensures it.
func (s *linuxShaper) Start(interfaceName string) error {
	applyLimits := func() error {
		s.ws.Clear(interfaceName)

		if config.GetBool(config.FlagShaperEnabled) {
			err := s.ws.LimitDownlink(interfaceName, int(config.GetUInt64(config.FlagShaperBandwidth))*8)
			if err != nil {
				log.Error().Err(err).Msg("Could not limit download speed")
				return err
			}
			err = s.ws.LimitUplink(interfaceName, int(config.GetUInt64(config.FlagShaperBandwidth))*8)
			if err != nil {
				log.Error().Err(err).Msg("Could not limit upload speed")
				return err
			}
		}
		return nil
	}

	err := s.listener.SubscribeAsync(s.listenTopic, applyLimits)
	if err != nil {
		return errors.Wrap(err, "could not subscribe to topic: "+s.listenTopic)
	}

	return applyLimits()
}

// Clear clears shaping rules.
func (s *linuxShaper) Clear(interfaceName string) {
	s.ws.Clear(interfaceName)
}
