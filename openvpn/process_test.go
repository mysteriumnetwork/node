/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package openvpn

import (
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/openvpn/config"
	"github.com/stretchr/testify/assert"
)

func TestOpenvpnProcessStartsAndStopsSuccessfully(t *testing.T) {
	process := newProcess("testdata/openvpn-mock-client.sh", &config.GenericConfig{})

	err := process.Start()
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	process.Stop()

	err = process.Wait()
	assert.NoError(t, err)
}

func TestOpenvpnProcessStartReportsErrorIfCmdWrapperDiesTooEarly(t *testing.T) {
	process := newProcess("testdata/failing-openvpn-mock-client.sh", &config.GenericConfig{})

	err := process.Start()
	assert.Error(t, err)
}
