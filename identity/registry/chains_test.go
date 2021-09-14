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

package registry

import (
	"fmt"
	"testing"

	"github.com/mysteriumnetwork/node/config"
	"github.com/stretchr/testify/assert"
)

func Test_AvailableChains(t *testing.T) {
	chains := Chains()
	for _, id := range []int64{
		config.FlagChain1ChainID.Value,
		config.FlagChain2ChainID.Value,
	} {
		t.Run(fmt.Sprintf("ChainID: %d must be defined", id), func(t *testing.T) {
			_, ok := chains[id]
			assert.True(t, ok, fmt.Sprintf("ChainID: %d is not defined", id))
		})
	}
}
