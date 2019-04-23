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

package location

import (
	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
)

const fallbackResolverLogPrefix = "[fallback-resolver] "

// ErrLocationResolutionFailed represents a failure to resolve location and running out of fallbacks to try
var ErrLocationResolutionFailed = errors.New("location resolution failed")

// FallbackResolver represents a resolver that tries multiple resolution techniques in sequence until one of them completes successfully, or all of them fail.
type FallbackResolver struct {
	LocationResolvers []Resolver
}

// NewFallbackResolver returns a new instance of fallback resolver
func NewFallbackResolver(resolvers []Resolver) *FallbackResolver {
	return &FallbackResolver{
		LocationResolvers: resolvers,
	}
}

// DetectLocation allows us to detect our current location
func (fr *FallbackResolver) DetectLocation() (Location, error) {
	for _, v := range fr.LocationResolvers {
		loc, err := v.DetectLocation()
		if err != nil {
			log.Warn(fallbackResolverLogPrefix, "could not resolve location", err)
		} else {
			return loc, err
		}
	}
	return Location{}, ErrLocationResolutionFailed
}
