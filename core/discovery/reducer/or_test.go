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

package reducer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Or_FiltersAlwaysMatching(t *testing.T) {
	match := Or(conditionAlwaysMatch, conditionAlwaysMatch)

	assert.True(t, match(proposalEmpty))
	assert.True(t, match(proposalEmpty))
}

func Test_Or_SkipsNeverMatching(t *testing.T) {
	match := Or(conditionAlwaysMatch, conditionNeverMatch)

	assert.True(t, match(proposalEmpty))
	assert.True(t, match(proposalEmpty))
}

func Test_Or_FiltersStreamingOrProvider(t *testing.T) {
	match := Or(conditionIsProvider1, conditionIsStreaming)

	assert.False(t, match(proposalEmpty))
	assert.True(t, match(proposalProvider1Streaming))
	assert.True(t, match(proposalProvider1Noop))
	assert.True(t, match(proposalProvider2Streaming))
}
