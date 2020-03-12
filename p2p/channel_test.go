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
	"net"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/stretchr/testify/assert"
)

func TestChannelFullCommunicationFlow(t *testing.T) {
	var err error
	var consumer, provider *Channel
	ports := acquirePorts(t, 2)
	providerPort := ports[0]
	consumerPort := ports[1]
	privateKey := GeneratePrivateKey()
	assert.NoError(t, err)

	t.Run("Test provider channel creation", func(t *testing.T) {
		provider, err = NewChannel(providerPort, privateKey)
		assert.NoError(t, err)
		go func() {
			err = provider.ListenAndServe()
			assert.NoError(t, err)
		}()
	})

	t.Run("Test consumer channel creation", func(t *testing.T) {
		consumer, err = NewChannel(consumerPort, privateKey)
		assert.NoError(t, err)
		go func() {
			err := consumer.ListenAndServe()
			assert.NoError(t, err)
		}()
	})

	t.Run("Test peer join", func(t *testing.T) {
		assert.NotNil(t, consumer)
		consumer.JoinPeer(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: providerPort})

		assert.NotNil(t, provider)
		provider.JoinPeer(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: consumerPort})
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
		_, err = consumer.Send("ping.pong", msg)
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
		assert.NoError(t, err)

		msg := ProtoMessage(&pb.PingPong{Value: "ping"})
		assert.NoError(t, err)
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

	// Create provider.
	provider, err := NewChannel(ports[0], GeneratePrivateKey())
	assert.NoError(t, err)
	provider.Handle("test", func(c Context) error {
		return c.OkWithReply(&Message{Data: []byte("hello")})
	})
	go provider.ListenAndServe()

	// Create consumer with his own private key. Send should timeout.
	consumer, err := NewChannel(ports[1], GeneratePrivateKey())
	assert.NoError(t, err)
	consumer.JoinPeer(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: ports[0]})
	consumer.SetSendTimeout(10 * time.Millisecond)
	_, err = consumer.Send("test", &Message{Data: []byte("hello")})
	assert.Error(t, err)
}

func TestChannelSendReturnErrorWhenPeerCannotHandleIt(t *testing.T) {
	ports := acquirePorts(t, 2)
	key := GeneratePrivateKey()
	// Create provider.
	provider, err := NewChannel(ports[0], key)
	assert.NoError(t, err)
	provider.Handle("test", func(c Context) error {
		return c.Error(errors.New("I don't like you"))
	})
	go provider.ListenAndServe()

	// Create consumer with his own private key. Send should timeout.
	consumer, err := NewChannel(ports[1], key)
	assert.NoError(t, err)
	consumer.JoinPeer(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: ports[0]})
	_, err = consumer.Send("test", &Message{Data: []byte("hello")})
	assert.EqualError(t, err, "could not send message: send failed: I don't like you")
}

func TestChannelListenFailWithInvalidKey(t *testing.T) {
	_, err := NewChannel(1234, []byte("invalid"))
	assert.Error(t, err)
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
