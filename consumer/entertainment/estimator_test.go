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

package entertainment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEstimator(t *testing.T) {
	// given
	estimator := NewEstimator(0.01, 0.0001)

	// expect
	assert.Equal(t, 102400.0, estimator.totalTrafficMiB(1))
	assert.Equal(t, 204800.0, estimator.totalTrafficMiB(2))

	assert.Equal(t, uint64(106), estimator.minutes(1, 1000))
	assert.Equal(t, uint64(53), estimator.minutes(1, 2000))
	assert.Equal(t, uint64(9555), estimator.minutes(1, 0.5))

	// and
	e := estimator.EstimatedEntertainment(5)
	assert.Equal(t, uint64(536870), e.TrafficMB)
	assert.Equal(t, 0.01, e.PricePerGiB)
	assert.Equal(t, 0.0001, e.PricePerMin)

	// can fluctuate based on constants so just assert it's set
	assert.Less(t, uint64(0), e.VideoMinutes)
	assert.Less(t, uint64(0), e.MusicMinutes)
	assert.Less(t, uint64(0), e.BrowsingMinutes)
}
