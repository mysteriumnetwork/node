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

package router

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_router_ExcludeIP(t *testing.T) {
	tests := []struct {
		name            string
		ips             []net.IP
		expectedRecords int
		wantErr         bool
	}{
		{
			name:            "Adding multiple unique records",
			ips:             []net.IP{net.ParseIP("1.2.3.4"), net.ParseIP("2.3.4.5"), net.ParseIP("3.4.5.6")},
			expectedRecords: 3,
			wantErr:         false,
		},
		{
			name:            "Adding duplicated rules saves only once",
			ips:             []net.IP{net.ParseIP("1.2.3.4"), net.ParseIP("1.2.3.4")},
			expectedRecords: 1,
			wantErr:         false,
		},
		{
			name:            "No panic on empty IP, just expected error",
			ips:             []net.IP{nil},
			expectedRecords: 1,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := &mockRoutingTable{gw: net.ParseIP("1.1.1.1")}
			r := &manager{
				stop:         make(chan struct{}),
				routingTable: table,
			}

			for _, ip := range tt.ips {
				if err := r.ExcludeIP(ip); (err != nil) != tt.wantErr {
					t.Errorf("Error = %v, wantErr %v", err, tt.wantErr)
				}
			}

			assert.Len(t, table.rules, tt.expectedRecords, "Expected number of table rules does not match")
		})
	}
}

func Test_router_Clean(t *testing.T) {
	tests := []struct {
		name            string
		ips             []net.IP
		expectedRecords int
		wantErr         bool
	}{
		{
			name:            "Clean multiple unique records",
			ips:             []net.IP{net.ParseIP("1.2.3.4"), net.ParseIP("2.3.4.5"), net.ParseIP("3.4.5.6")},
			expectedRecords: 0,
			wantErr:         false,
		},
		{
			name:            "Clean duplicated rules only once",
			ips:             []net.IP{net.ParseIP("1.2.3.4"), net.ParseIP("1.2.3.4")},
			expectedRecords: 0,
			wantErr:         false,
		},
		{
			name:            "No panic on empty table, just expected error",
			ips:             []net.IP{},
			expectedRecords: 0,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := &mockRoutingTable{gw: net.ParseIP("1.1.1.1")}
			r := &manager{
				stop:         make(chan struct{}),
				routingTable: table,
			}

			for _, ip := range tt.ips {
				if err := r.ExcludeIP(ip); (err != nil) != tt.wantErr {
					t.Errorf("Error = %v, wantErr %v", err, tt.wantErr)
				}
			}

			err := r.Clean()

			assert.NoError(t, err)

			assert.Len(t, table.rules, tt.expectedRecords, "Expected number of table rules does not match")
		})
	}
}

func Test_router_ReplaceGW(t *testing.T) {
	table := &mockRoutingTable{gw: net.ParseIP("1.1.1.1")}
	r := &manager{
		stop:            make(chan struct{}),
		gwCheckInterval: 100 * time.Millisecond,
		routingTable:    table,
	}

	r.ExcludeIP(net.ParseIP("2.2.2.2"))
	r.ExcludeIP(net.ParseIP("3.3.3.3"))

	assert.Contains(t, table.rules, "2.2.2.2:1.1.1.1")
	assert.Contains(t, table.rules, "3.3.3.3:1.1.1.1")
	assert.NotContains(t, table.rules, "2.2.2.2:4.4.4.4")
	assert.NotContains(t, table.rules, "3.3.3.3:4.4.4.4")
	assert.Len(t, table.rules, 2)

	table.setGW(net.ParseIP("4.4.4.4"))

	assert.Eventually(t, func() bool {
		table.mu.Lock()
		defer table.mu.Unlock()

		_, ok1 := table.rules["2.2.2.2:4.4.4.4"]
		_, ok2 := table.rules["3.3.3.3:4.4.4.4"]
		_, not1 := table.rules["2.2.2.2:1.1.1.1"]
		_, not2 := table.rules["3.3.3.3:1.1.1.1"]

		return ok1 && ok2 && !not1 && !not2
	}, time.Second, 10*time.Millisecond)

	assert.Len(t, table.rules, 2)
}

type mockRoutingTable struct {
	rules map[string]int
	gw    net.IP

	mu sync.Mutex
}

func (t *mockRoutingTable) ExcludeRule(ip, gw net.IP) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.rules == nil {
		t.rules = make(map[string]int)
	}

	t.rules[fmt.Sprintf("%s:%s", ip, gw)]++

	if ip.Equal(nil) {
		return fmt.Errorf("expected error")
	}

	return nil
}

func (t *mockRoutingTable) DeleteRule(ip, gw net.IP) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.rules == nil {
		t.rules = make(map[string]int)
	}

	t.rules[fmt.Sprintf("%s:%s", ip, gw)]--

	if t.rules[fmt.Sprintf("%s:%s", ip, gw)] == 0 {
		delete(t.rules, fmt.Sprintf("%s:%s", ip, gw))
	}

	return nil
}

func (t *mockRoutingTable) DiscoverGateway() (net.IP, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.gw, nil
}

func (t *mockRoutingTable) setGW(gw net.IP) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.gw = gw
}
