/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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
	"math/big"
	"time"

	"github.com/mysteriumnetwork/node/identity"
)

// NewStats initiates zero Stats instance.
func NewStats() Stats {
	return Stats{
		ConsumerCounts: make(map[identity.Identity]int),
		SumTokens:      new(big.Int),
	}
}

// Stats holds structure of aggregate session statistics.
type Stats struct {
	Count           int
	ConsumerCounts  map[identity.Identity]int
	SumDataSent     uint64
	SumDataReceived uint64
	SumDuration     time.Duration
	SumTokens       *big.Int
}

// Add accumulates given session to statistics.
func (s *Stats) Add(session History) {
	s.Count++

	if _, found := s.ConsumerCounts[session.ConsumerID]; !found {
		s.ConsumerCounts[session.ConsumerID] = 1
	} else {
		s.ConsumerCounts[session.ConsumerID]++
	}

	s.SumDataReceived += session.DataReceived
	s.SumDataSent += session.DataSent
	s.SumDuration += session.GetDuration()
	s.SumTokens = new(big.Int).Add(s.SumTokens, session.Tokens)
}
