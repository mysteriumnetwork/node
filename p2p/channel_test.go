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
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/pb"
)

func TestChannelFullCommunicationFlow(t *testing.T) {
	provider, consumer, err := createTestChannels()
	require.NoError(t, err)
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
		_, err := consumer.Send(context.Background(), "ping.pong", msg)
		assert.NoError(t, err)

		publishedProviderMsg := &pb.PingPong{Value: "Provider SmallZ"}
		msg = ProtoMessage(publishedProviderMsg)
		_, err = provider.Send(context.Background(), "ping.pong", msg)
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
		res, err := consumer.Send(context.Background(), "testreq", msg)
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
				_, err := consumer.Send(context.Background(), "concurrent", &Message{Data: []byte{}})
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
			consumer.Send(context.Background(), "slow", &Message{})
		}()

		fastFinished := make(chan struct{})
		go func() {
			<-slowStarted
			consumer.Send(context.Background(), "fast", &Message{})
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

		_, err := consumer.Send(context.Background(), "get-error", &Message{Data: []byte("hello")})
		assert.EqualError(t, err, "public peer error: I don't like you")
	})

	t.Run("Test peer returns internal error", func(t *testing.T) {
		provider.Handle("get-error", func(c Context) error {
			return errors.New("I don't like you")
		})

		_, err := consumer.Send(context.Background(), "get-error", &Message{Data: []byte("hello")})
		assert.EqualError(t, err, "peer error: I don't like you")
	})

	t.Run("Test peer returns handler not found error", func(t *testing.T) {
		_, err := consumer.Send(context.Background(), "ping", &Message{Data: []byte("hello")})
		if !errors.Is(err, ErrHandlerNotFound) {
			t.Fatalf("expect handler not found err, got %v", err)
		}
	})
}

func TestChannel_Send_Timeout(t *testing.T) {
	provider, consumer, err := createTestChannels()
	require.NoError(t, err)
	defer provider.Close()
	defer consumer.Close()

	t.Run("Test timeout for long not responding peer", func(t *testing.T) {
		provider.Handle("timeout", func(c Context) error {
			time.Sleep(time.Hour)
			return c.OK()
		})

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, err = consumer.Send(ctx, "timeout", &Message{Data: []byte("ping")})
		if !errors.Is(err, ErrSendTimeout) {
			t.Fatalf("expect timeout err, got: %v", err)
		}
	})

	t.Run("Test timeout when peer is closed", func(t *testing.T) {
		provider.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, err = consumer.Send(ctx, "timeout", &Message{Data: []byte("ping")})
		if !errors.Is(err, ErrSendTimeout) {
			t.Fatalf("expect timeout err, got: %v", err)
		}
	})
}

func TestChannel_Send_To_When_Peer_Starts_Later(t *testing.T) {
	provider, consumer, err := createTestChannels()
	require.NoError(t, err)
	defer consumer.Close()
	defer provider.Close()

	// Close provider channel to simulate unstable network.
	// Consumer will try to send messages and during first 50 ms
	// they will not reach provider peer, but since kcp will try
	// keep resending packets they will finally reach opened
	// provider peer.
	addr := provider.(*channel).tr.remoteConn.LocalAddr().(*net.UDPAddr)
	err = provider.Close()
	require.NoError(t, err)
	go func() {
		time.Sleep(50 * time.Millisecond)
		provider, err := reopenChannel(provider.(*channel), addr)

		require.NoError(t, err)
		provider.Handle("timeout", func(c Context) error {
			return c.OK()
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err = consumer.Send(ctx, "timeout", &Message{Data: []byte("ping")})
	require.NoError(t, err)
}

func TestChannel_Detect_And_Update_Peer_Addr(t *testing.T) {
	provider, consumer, err := createTestChannels()
	require.NoError(t, err)
	defer consumer.Close()
	defer provider.Close()

	provider.Handle("ping", func(c Context) error {
		return c.OK()
	})

	// Close consumer peer and reopen channel with new local addr.
	consumer.Close()
	consumer, err = reopenChannelWithNewLocalAddr(consumer.(*channel))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = consumer.Send(ctx, "ping", &Message{Data: []byte("pingasssas")})
}

func BenchmarkChannel_Send(b *testing.B) {
	provider, consumer, err := createTestChannels()
	require.NoError(b, err)
	defer provider.Close()
	defer consumer.Close()

	provider.Handle("bench", func(c Context) error {
		return c.OkWithReply(&Message{Data: []byte("I'm still OK")})
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := consumer.Send(context.Background(), "bench", &Message{Data: []byte("Catch this!")})
		require.NoError(b, err)
		require.NotNil(b, res)
	}
}

func reopenChannel(c *channel, addr *net.UDPAddr) (*channel, error) {
	punchedConn, err := net.DialUDP("udp4", addr, c.peer.addr())
	if err != nil {
		return nil, err
	}
	ch, err := newChannel(punchedConn, c.privateKey, c.peer.publicKey, 1)
	if err != nil {
		return nil, err
	}
	ch.launchReadSendLoops()
	return ch, err
}

func reopenChannelWithNewLocalAddr(c *channel) (*channel, error) {
	punchedConn, err := net.DialUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")}, c.peer.addr())
	if err != nil {
		return nil, err
	}
	ch, err := newChannel(punchedConn, c.privateKey, c.peer.publicKey, 1)
	if err != nil {
		return nil, err
	}
	ch.launchReadSendLoops()
	return ch, err
}

func createTestChannels() (Channel, Channel, error) {
	ports, err := acquirePorts(2)
	if err != nil {
		return nil, nil, err
	}
	providerPort := ports[0]
	consumerPort := ports[1]

	providerConn, err := net.DialUDP("udp4", &net.UDPAddr{Port: providerPort}, &net.UDPAddr{Port: consumerPort})
	if err != nil {
		return nil, nil, err
	}

	consumerConn, err := net.DialUDP("udp4", &net.UDPAddr{Port: consumerPort}, &net.UDPAddr{Port: providerPort})
	if err != nil {
		return nil, nil, err
	}

	providerPublicKey, providerPrivateKey, err := GenerateKey()
	if err != nil {
		return nil, nil, err
	}
	consumerPublicKey, consumerPrivateKey, err := GenerateKey()
	if err != nil {
		return nil, nil, err
	}

	provider, err := newChannel(providerConn, providerPrivateKey, consumerPublicKey, 1)
	if err != nil {
		return nil, nil, err
	}
	provider.launchReadSendLoops()

	consumer, err := newChannel(consumerConn, consumerPrivateKey, providerPublicKey, 1)
	if err != nil {
		return nil, nil, err
	}
	consumer.launchReadSendLoops()

	return provider, consumer, nil
}

func acquirePorts(n int) ([]int, error) {
	portPool := port.NewFixedRangePool(port.Range{Start: 10000, End: 60000})
	ports, err := portPool.AcquireMultiple(n)
	if err != nil {
		return nil, err
	}
	var res []int
	for _, v := range ports {
		res = append(res, v.Num())
	}
	return res, nil
}
