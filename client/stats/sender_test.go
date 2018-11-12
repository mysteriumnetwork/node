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

package stats

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server"
	"github.com/stretchr/testify/assert"
)

func TestRemoteStatsSenderOnDisconnect(t *testing.T) {
	var counter int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
	}))
	defer ts.Close()

	statsKeeper := NewSessionStatsKeeper(time.Now)
	mysteriumClient := server.NewClient(ts.URL)
	signer := &identity.SignerFake{}
	sender := NewRemoteStatsSender(statsKeeper, mysteriumClient, "0x00000", identity.Identity{Address: "0x00001"}, "openvpn", signer, "KG", time.Minute)

	sender.StateHandler(openvpn.ConnectedState)
	assert.NoError(t, waitFor(func() bool { return atomic.LoadInt64(&counter) == 0 }))

	sender.StateHandler(openvpn.ExitingState)
	assert.NoError(t, waitFor(func() bool { return atomic.LoadInt64(&counter) == 1 }))
}

func TestRemoteStatsSenderInterval(t *testing.T) {
	var counter int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
	}))
	defer ts.Close()

	signer := &identity.SignerFake{}
	mysteriumClient := server.NewClient(ts.URL)
	statsKeeper := NewSessionStatsKeeper(time.Now)
	sender := NewRemoteStatsSender(statsKeeper, mysteriumClient, "0x00000", identity.Identity{Address: "0x00001"}, "openvpn", signer, "KG", time.Nanosecond)

	sender.StateHandler(openvpn.ConnectedState)
	assert.NoError(t, waitFor(func() bool { return atomic.LoadInt64(&counter) > 3 }))

	sender.StateHandler(openvpn.ExitingState)
}

func waitFor(f func() bool) error {
	timeout := time.Now().Add(time.Second)
	for time.Now().Before(timeout) {
		if f() {
			return nil
		}
	}
	return fmt.Errorf("Failed to wait for expected result")
}
