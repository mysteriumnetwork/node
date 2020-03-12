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

package p2p_test

import (
	"errors"
	"net"

	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
)

func ExampleChannel_Handle() {
	ch, err := p2p.NewChannel(1234, p2p.GeneratePrivateKey())
	if err != nil {
		// handle error ...
		return
	}
	defer ch.Close()

	// Simple handler without response data.
	ch.Handle("ping", func(c p2p.Context) error {
		return c.OK()
	})

	// Handler which reads request and sends reply.
	ch.Handle("ping-pong", func(c p2p.Context) error {
		var ping pb.PingPong
		if err := c.Request().UnmarshalProto(&ping); err != nil {
			return err
		}

		res := p2p.ProtoMessage(&pb.PingPong{
			Value: ping.Value + "pong",
		})

		return c.OkWithReply(res)
	})

	// Handler which returns error which will be visible for another peer.
	ch.Handle("no-ping-for-you", func(c p2p.Context) error {
		return c.Error(errors.New("I don't know you"))
	})

	// Handler which return internal error. In such case peer will get 500 error.
	ch.Handle("no-ping-for-you", func(c p2p.Context) error {
		return errors.New("ups, some internal error")
	})
}

func ExampleChannel_JoinPeer() {
	ch, err := p2p.NewChannel(1234, p2p.GeneratePrivateKey())
	if err != nil {
		// handle error ...
		return
	}

	ch.JoinPeer(&net.UDPAddr{IP: net.ParseIP("peer ip"), Port: 12345})
}

func ExampleChannel_Send() {
	ch, err := p2p.NewChannel(1234, p2p.GeneratePrivateKey())
	if err != nil {
		// handle error ...
		return
	}

	// Send request and ignore response if not needed.
	_, err = ch.Send("ping", p2p.ProtoMessage(&pb.PingPong{Value: "ping"}))

	// Send requests and get for response data.
	resMsg, err := ch.Send("ping", p2p.ProtoMessage(&pb.PingPong{Value: "ping"}))
	if err != nil {
		// handle error ...
		return
	}
	var pong pb.PingPong
	if err := resMsg.UnmarshalProto(&pong); err != nil {
		// handle error ...
	}
}

func ExampleChannel_ListenAndServe() {
	ch, err := p2p.NewChannel(1234, p2p.GeneratePrivateKey())
	if err != nil {
		// handle error ...
		return
	}

	// Start listening.
	go func() {
		if err := ch.ListenAndServe(); err != nil {
			// handle error ...
		}
	}()
}
