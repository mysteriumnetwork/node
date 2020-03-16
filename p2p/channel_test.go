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

package p2p

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/stretchr/testify/assert"
)

func TestChannelFullCommunicationFlow(t *testing.T) {
	var provider, consumer *Channel
	ports := acquirePorts(t, 2)
	providerPort := ports[0]
	consumerPort := ports[1]

	providerPublicKey, providerPrivateKey, err := GenerateKey()
	assert.NoError(t, err)
	consumerPublicKey, consumerPrivateKey, err := GenerateKey()
	assert.NoError(t, err)

	t.Run("Test provider channel creation", func(t *testing.T) {
		peer := &Peer{
			Addr:      &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: consumerPort},
			PublicKey: consumerPublicKey,
		}
		var err error
		provider, err = NewChannel(providerPort, providerPrivateKey, peer)
		assert.NoError(t, err)
	})

	t.Run("Test consumer channel creation", func(t *testing.T) {
		peer := &Peer{
			Addr:      &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: providerPort},
			PublicKey: providerPublicKey,
		}
		var err error
		consumer, err = NewChannel(consumerPort, consumerPrivateKey, peer)
		assert.NoError(t, err)
	})

	t.Run("Test publish subscribe pattern", func(t *testing.T) {
		consumerReceivedMsg := make(chan *pb.PingPong, 1)
		providerReceivedMsg := make(chan *pb.PingPong, 1)

		consumer.Handle("ping.pong", func(c Context) error {
			var res pb.PingPong
			err := c.Request().UnmarshalProto(&res)
			assert.NoError(t, err)
			consumerReceivedMsg <- &res
			return c.OK()
		})

		provider.Handle("ping.pong", func(c Context) error {
			var res pb.PingPong
			err := c.Request().UnmarshalProto(&res)
			assert.NoError(t, err)
			providerReceivedMsg <- &res
			return c.OK()
		})

		publishedConsumerMsg := &pb.PingPong{Value: "Consumer BigZ"}
		msg := ProtoMessage(publishedConsumerMsg)
		_, err := consumer.Send("ping.pong", msg)
		assert.NoError(t, err)

		publishedProviderMsg := &pb.PingPong{Value: "Provider SmallZ"}
		msg = ProtoMessage(publishedProviderMsg)
		_, err = provider.Send("ping.pong", msg)
		assert.NoError(t, err)

		select {
		case v := <-consumerReceivedMsg:
			assert.Equal(t, publishedProviderMsg.Value, v.Value)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("did not received message from channel consumer subscription")
		}

		select {
		case v := <-providerReceivedMsg:
			assert.Equal(t, publishedConsumerMsg.Value, v.Value)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("did not received message from channel provider subscription")
		}
	})

	t.Run("Test request reply pattern", func(t *testing.T) {
		provider.Handle("testreq", func(c Context) error {
			var req pb.PingPong
			err := c.Request().UnmarshalProto(&req)
			assert.NoError(t, err)

			msg := ProtoMessage(&pb.PingPong{Value: req.Value + "-pong"})
			assert.NoError(t, err)
			return c.OkWithReply(msg)
		})

		msg := ProtoMessage(&pb.PingPong{Value: "ping"})
		res, err := consumer.Send("testreq", msg)
		assert.NoError(t, err)

		var resMsg pb.PingPong
		err = res.UnmarshalProto(&resMsg)
		assert.NoError(t, err)
		assert.Equal(t, "ping-pong", resMsg.Value)
	})
}

func TestChannelSendTimeoutWhenPrivateKeysMismatch(t *testing.T) {
	ports := acquirePorts(t, 2)

	// Create provider consumer keys.
	_, providerPrivateKey, err := GenerateKey()
	assert.NoError(t, err)
	consumerPublicKey, consumerPrivateKey, err := GenerateKey()
	assert.NoError(t, err)

	// Create provider.
	provider, err := NewChannel(ports[0], providerPrivateKey, &Peer{
		Addr:      &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: ports[1]},
		PublicKey: consumerPublicKey,
	})
	assert.NoError(t, err)
	provider.Handle("test", func(c Context) error {
		return c.OkWithReply(&Message{Data: []byte("hello")})
	})

	// Create consumer with incorrect providers public key. Send should timeout.
	consumer, err := NewChannel(ports[1], consumerPrivateKey, &Peer{
		Addr:      &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: ports[0]},
		PublicKey: consumerPublicKey, // For correct setup here should be provider key.
	})
	assert.NoError(t, err)
	consumer.SetSendTimeout(300 * time.Millisecond)
	_, err = consumer.Send("test", &Message{Data: []byte("hello")})
	assert.EqualError(t, err, fmt.Sprintf("could not send message: Post http://127.0.0.1:%d/test: context deadline exceeded", ports[0]))
}

func TestChannelSendReturnErrorWhenPeerCannotHandleIt(t *testing.T) {
	ports := acquirePorts(t, 2)

	// Create provider consumer keys.
	providerPublicKey, providerPrivateKey, err := GenerateKey()
	assert.NoError(t, err)
	consumerPublicKey, consumerPrivateKey, err := GenerateKey()
	assert.NoError(t, err)

	// Create provider.
	provider, err := NewChannel(ports[0], providerPrivateKey, &Peer{
		Addr:      &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: ports[1]},
		PublicKey: consumerPublicKey,
	})
	assert.NoError(t, err)
	provider.Handle("test", func(c Context) error {
		return c.Error(errors.New("I don't like you"))
	})

	// Create consumer with incorrect providers public key. Send should timeout.
	consumer, err := NewChannel(ports[1], consumerPrivateKey, &Peer{
		Addr:      &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: ports[0]},
		PublicKey: providerPublicKey,
	})
	assert.NoError(t, err)
	consumer.SetSendTimeout(300 * time.Millisecond)
	_, err = consumer.Send("test", &Message{Data: []byte("hello")})
	assert.EqualError(t, err, "could not send message: peer error response: I don't like you")
}

func acquirePorts(t *testing.T, n int) []int {
	portPool := port.NewPool()
	ports, err := portPool.AcquireMultiple(n)
	assert.NoError(t, err)
	var res []int
	for _, v := range ports {
		res = append(res, v.Num())
	}
	return res
}
