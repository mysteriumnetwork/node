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

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/session"
)

type keepAliveTestChannel struct {
	p2p.Channel
	handler p2p.HandlerFunc
}

func (ch *keepAliveTestChannel) Handle(topic string, handler p2p.HandlerFunc) {
	if topic == p2p.TopicKeepAlive {
		ch.handler = handler
	}
}

type keepAliveTestContext struct {
	p2p.Context
	request      *p2p.Message
	peer         identity.Identity
	publicError  error
	acknowledged bool
}

func (c *keepAliveTestContext) Request() *p2p.Message     { return c.request }
func (c *keepAliveTestContext) PeerID() identity.Identity { return c.peer }
func (c *keepAliveTestContext) Error(err error) error     { c.publicError = err; return nil }
func (c *keepAliveTestContext) OK() error                 { c.acknowledged = true; return nil }

func TestSessionManager_KeepAliveValidatesSession(t *testing.T) {
	for _, tt := range []struct {
		name          string
		pingID        string
		otherConsumer bool
		closed        bool
		removed       bool
		malformed     bool
		rejected      bool
	}{
		{name: "running session", pingID: "active"},
		{name: "empty session ID"},
		{name: "unknown session", pingID: "unknown", rejected: true},
		{name: "other consumer", pingID: "active", otherConsumer: true, rejected: true},
		{name: "closed session still in storage", pingID: "active", closed: true, rejected: true},
		{name: "closed session removed", pingID: "active", closed: true, removed: true, rejected: true},
		{name: "malformed ping", malformed: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewSessionPool(mocks.NewEventBus())
			active := &Session{ID: session.ID("active"), ConsumerID: consumerID, done: make(chan struct{})}
			pool.Add(active)
			defer active.Close()
			if tt.closed {
				active.Close()
			}
			if tt.removed {
				pool.Remove(active.ID)
			}
			manager := &SessionManager{sessionStorage: pool, config: Config{KeepAlive: KeepAliveConfig{SendInterval: time.Hour}}}
			ch := &keepAliveTestChannel{}
			// Register the handler with a stopped loop, as it remains registered after closure.
			stopped := &Session{done: make(chan struct{})}
			stopped.Close()
			manager.keepAliveLoop(stopped, ch)
			require.NotNil(t, ch.handler)
			ctx := &keepAliveTestContext{request: p2p.ProtoMessage(&pb.P2PKeepAlivePing{SessionID: tt.pingID}), peer: consumerID}
			if tt.otherConsumer {
				ctx.peer = identity.FromAddress("0x1234")
			}
			if tt.malformed {
				ctx.request = &p2p.Message{Data: []byte{0xff}}
			}
			err := ch.handler(ctx)
			if tt.malformed {
				require.Error(t, err)
				require.False(t, ctx.acknowledged)
				return
			}
			require.NoError(t, err)
			if tt.rejected {
				require.EqualError(t, ctx.publicError, "session "+tt.pingID+" is not active")
				require.False(t, ctx.acknowledged)
			} else {
				require.NoError(t, ctx.publicError)
				require.True(t, ctx.acknowledged)
			}
		})
	}
}
