/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package netstack_provider

import (
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

const AppTopicConfigShaper = "config:shaper"

var rateLimiter *rate.Limiter

func getRateLimitter() *rate.Limiter {
	return rateLimiter
}

func InitUserspaceShaper(eventBus eventbus.EventBus) {
	applyLimits := func(e interface{}) {
		bandwidthBytes := config.GetUInt64(config.FlagShaperBandwidth) * 1024
		bandwidth := rate.Limit(bandwidthBytes)
		if !config.GetBool(config.FlagShaperEnabled) {
			bandwidth = rate.Inf
		}
		log.Info().Msgf("Shaper bandwidth: %v", config.GetUInt64(config.FlagShaperBandwidth))
		rateLimiter.SetLimit(bandwidth)
	}

	rateLimiter = rate.NewLimiter(rate.Inf, 0)
	applyLimits(nil)

	err := eventBus.SubscribeAsync(AppTopicConfigShaper, applyLimits)
	if err != nil {
		log.Error().Msgf("could not subscribe to topic: %v", err)
	}
}
