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

package discovery

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockDiscovery struct {
	err     error
	started bool
	stopped bool
}

func (md *mockDiscovery) Start() error {
	md.started = true
	return md.err
}

func (md *mockDiscovery) Stop() error {
	md.stopped = true
	return md.err
}

func Test_multiDiscovery_StartsAll(t *testing.T) {
	d1 := &mockDiscovery{}
	d2 := &mockDiscovery{}

	md := multiDiscovery{
		bonjour: d1,
		ssdp:    d2,
	}

	err := md.Start()
	assert.NoError(t, err)

	assert.True(t, d1.started)
	assert.True(t, d2.started)
}

func Test_multiDiscovery_StartsWithError(t *testing.T) {
	d1 := &mockDiscovery{err: errors.New("error1")}
	d2 := &mockDiscovery{err: errors.New("error2")}

	md := multiDiscovery{
		bonjour: d1,
		ssdp:    d2,
	}

	err := md.Start()
	assert.EqualError(t, err, "error1")

	assert.True(t, d1.started)
	assert.False(t, d2.started)
}

func Test_multiDiscovery_StopsAll(t *testing.T) {
	d1 := &mockDiscovery{}
	d2 := &mockDiscovery{}

	md := multiDiscovery{
		bonjour: d1,
		ssdp:    d2,
	}

	err := md.Stop()
	assert.NoError(t, err)

	assert.True(t, d1.stopped)
	assert.True(t, d2.stopped)
}

func Test_multiDiscovery_StopsWithError(t *testing.T) {
	d1 := &mockDiscovery{err: errors.New("error1")}
	d2 := &mockDiscovery{err: errors.New("error2")}

	md := multiDiscovery{
		bonjour: d1,
		ssdp:    d2,
	}

	err := md.Stop()
	assert.EqualError(t, err, "error2")

	assert.True(t, d1.stopped)
	assert.True(t, d2.stopped)
}
