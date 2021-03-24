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

package nats

import (
	"net/url"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

func TestBrokerConnector(t *testing.T) {
	assert := assert.New(t)

	// given
	srv := server.New(&server.Options{Port: 44224})
	go srv.Start()
	defer srv.Shutdown()
	assert.True(srv.ReadyForConnections(2 * time.Second))
	connector := NewBrokerConnector(nil)

	// when
	conn, err := connector.Connect(&url.URL{
		Scheme: DefaultBrokerScheme,
		Host:   srv.Addr().String(),
	})
	// then
	assert.NoError(err)
	defer conn.Close()

	// when
	sub := make(chan *nats.Msg)
	_, err = conn.Subscribe("#random", func(msg *nats.Msg) {
		sub <- msg
	})
	assert.NoError(err)
	err = conn.Publish("#random", []byte("anybody there?"))

	// then
	assert.NoError(err)
	assert.Eventually(func() bool {
		select {
		case received := <-sub:
			assert.Equal("anybody there?", string(received.Data))
			assert.Equal("#random", received.Subject)
			return true
		default:
			return false
		}
	}, 5*time.Second, 200*time.Millisecond)
}
