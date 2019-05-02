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

package location

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/connection"
)

func TestProperCache_needsRefresh(t *testing.T) {
	type fields struct {
		lastFetched time.Time
		expiry      time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "returns true on zero time",
			want:   true,
			fields: fields{},
		},
		{
			name: "returns true if expired",
			want: true,
			fields: fields{
				lastFetched: time.Now().Add(time.Second * -59),
				expiry:      time.Minute * 1,
			},
		},
		{
			name: "returns false if updated recently",
			want: false,
			fields: fields{
				lastFetched: time.Now().Add(time.Second * -61),
				expiry:      time.Minute * 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ProperCache{
				lastFetched: tt.fields.lastFetched,
				expiry:      tt.fields.expiry,
			}
			if got := c.needsRefresh(); got != tt.want {
				t.Errorf("ProperCache.needsRefresh() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockResolver struct {
	called      bool
	errToReturn error
}

func (mr *mockResolver) DetectLocation() (Location, error) {
	mr.called = true
	return Location{}, mr.errToReturn
}

func TestCacheHandlesConnection_Connected(t *testing.T) {
	r := &mockResolver{}
	c := &ProperCache{
		expiry:           time.Second * 1,
		locationDetector: r,
	}
	c.HandleConnectionEvent(connection.StateEvent{State: connection.Connected})
	assert.True(t, r.called)
}

func TestCacheHandlesConnection_NotConnected(t *testing.T) {
	r := &mockResolver{}
	c := &ProperCache{
		expiry:           time.Second * 1,
		locationDetector: r,
	}
	c.HandleConnectionEvent(connection.StateEvent{State: connection.NotConnected})
	assert.True(t, r.called)
}

func TestCacheIgnoresOther(t *testing.T) {
	r := &mockResolver{}
	c := &ProperCache{
		expiry:           time.Second * 1,
		locationDetector: r,
	}
	c.HandleConnectionEvent(connection.StateEvent{State: connection.Reconnecting})
	assert.False(t, r.called)
}
