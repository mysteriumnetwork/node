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
	"sync"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/stretchr/testify/assert"
)

func TestChannelFullCommunicationFlow(t *testing.T) {
	provider, consumer := createTestChannels(t)

	defer provider.Close()
	defer consumer.Close()

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

	t.Run("Test concurrent requests", func(t *testing.T) {
		var wg sync.WaitGroup
		provider.Handle("concurrent", func(c Context) error {
			wg.Done()
			return c.OK()
		})

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				_, err := consumer.Send("concurrent", &Message{Data: []byte{}})
				assert.NoError(t, err)
			}()
		}

		wg.Wait()
	})

	t.Run("Test slow topicHandlers are not blocking", func(t *testing.T) {
		provider.Handle("slow", func(c Context) error {
			time.Sleep(time.Hour)
			return c.OK()
		})

		provider.Handle("fast", func(c Context) error {
			return c.OK()
		})

		slowStarted := make(chan struct{})
		go func() {
			slowStarted <- struct{}{}
			consumer.Send("slow", &Message{})
		}()

		fastFinished := make(chan struct{})
		go func() {
			<-slowStarted
			consumer.Send("fast", &Message{})
			fastFinished <- struct{}{}
		}()

		select {
		case <-fastFinished:
		case <-time.After(time.Second):
			t.Fatal("slow handler blocks concurrent send")
		}
	})

	t.Run("Test peer returns public error", func(t *testing.T) {
		provider.Handle("get-error", func(c Context) error {
			return c.Error(errors.New("I don't like you"))
		})

		_, err := consumer.Send("get-error", &Message{Data: []byte("hello")})
		assert.EqualError(t, err, "public peer error: I don't like you")
	})

	t.Run("Test peer returns internal error", func(t *testing.T) {
		provider.Handle("get-error", func(c Context) error {
			return errors.New("I don't like you")
		})

		_, err := consumer.Send("get-error", &Message{Data: []byte("hello")})
		assert.EqualError(t, err, "internal peer error")
	})

	t.Run("Test peer returns handler not found error", func(t *testing.T) {
		_, err := consumer.Send("ping", &Message{Data: []byte("hello")})
		assert.EqualError(t, err, "public peer error: handler \"ping\" is not registered")
	})
}

func createTestChannels(t *testing.T) (Channel, Channel) {
	ports := acquirePorts(t, 2)
	providerPort := ports[0]
	consumerPort := ports[1]

	providerConn, err := net.DialUDP("udp", &net.UDPAddr{Port: providerPort}, &net.UDPAddr{Port: consumerPort})
	assert.NoError(t, err)

	consumerConn, err := net.DialUDP("udp", &net.UDPAddr{Port: consumerPort}, &net.UDPAddr{Port: providerPort})
	assert.NoError(t, err)

	providerPublicKey, providerPrivateKey, err := GenerateKey()
	assert.NoError(t, err)
	consumerPublicKey, consumerPrivateKey, err := GenerateKey()
	assert.NoError(t, err)

	provider, err := newChannel(providerConn, providerPrivateKey, consumerPublicKey)
	assert.NoError(t, err)

	consumer, err := newChannel(consumerConn, consumerPrivateKey, providerPublicKey)
	assert.NoError(t, err)

	return provider, consumer
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
