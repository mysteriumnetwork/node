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

package endpoints

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/connection"
	nodeEvent "github.com/mysteriumnetwork/node/core/node/event"
	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/stretchr/testify/assert"
)

type mockStateProvider struct {
	stateToReturn stateEvent.State
}

func (msp *mockStateProvider) GetState() stateEvent.State {
	return msp.stateToReturn
}

func TestHandler_Stops(t *testing.T) {
	h := NewSSEHandler(&mockStateProvider{})

	wait := make(chan struct{})
	go func() {
		h.serve()
		wait <- struct{}{}
	}()

	h.stop()
	<-wait
}

func TestHandler_ConsumeNodeEvent_Stops(t *testing.T) {
	h := NewSSEHandler(&mockStateProvider{})
	me := nodeEvent.Payload{
		Status: nodeEvent.StatusStopped,
	}
	h.ConsumeNodeEvent(me)
	h.serve()
}

func TestHandler_ConsumeNodeEvent_Starts(t *testing.T) {
	h := NewSSEHandler(&mockStateProvider{})
	me := nodeEvent.Payload{
		Status: nodeEvent.StatusStarted,
	}

	h.ConsumeNodeEvent(me)

	// without starting, this would block forever
	h.newClients <- make(chan string)
	h.newClients <- make(chan string)

	h.stop()
}

func TestHandler_SendsInitialAndFollowingStates(t *testing.T) {
	msp := &mockStateProvider{}
	h := NewSSEHandler(msp)
	go h.serve()
	defer h.stop()
	laddr := net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	listener, err := net.ListenTCP("tcp4", &laddr)
	assert.Nil(t, err)
	addr := listener.Addr()
	port := addr.(*net.TCPAddr).Port
	defer listener.Close()

	router := httprouter.New()
	router.GET("/whatever", h.Sub)
	serveExit := make(chan error)
	go func() {
		err = http.Serve(listener, router)
		serveExit <- err
	}()

	time.Sleep(time.Millisecond * 50)
	w := fmt.Sprintf("http://127.0.0.1:%v/whatever", port)
	req, _ := http.NewRequest("GET", w, nil)
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	c := http.Client{}
	resp, err := c.Do(req)
	assert.Nil(t, err)
	results := make(chan string)
	go func() {
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}
			stringified := strings.Join(strings.Fields(strings.TrimSpace(string(line))), " ")
			if len(stringified) > 0 {
				results <- stringified
			}
		}
	}()

	msg := <-results
	assert.Regexp(t, "^data:\\s?{.*}$", msg)
	msgJSON := strings.TrimPrefix(msg, "data: ")
	expectJSON := `
{
  "payload": {
    "nat_status": {
      "status": "",
      "error": ""
    },
    "service_info": null,
    "sessions": null,
    "consumer": {
      "connection": {
        "status": ""
      }
    },
    "identities": []
  },
  "type": "state-change"
}`
	assert.JSONEq(t, expectJSON, msgJSON)

	changedState := msp.GetState()
	changedState.NATStatus = stateEvent.NATStatus{
		Status: "mass panic",
		Error:  "cookie prices rise drastically",
	}
	h.ConsumeStateEvent(changedState)

	msg = <-results
	assert.Regexp(t, "^data:\\s?{.*}$", msg)
	msgJSON = strings.TrimPrefix(msg, "data: ")
	expectJSON = `
{
  "payload": {
    "nat_status": {
      "status": "mass panic",
      "error": "cookie prices rise drastically"
    },
    "service_info": null,
    "sessions": null,
    "consumer": {
      "connection": {
        "status": ""
      }
    },
    "identities": []
  },
  "type": "state-change"
}`
	assert.JSONEq(t, expectJSON, msgJSON)

	changedState = msp.GetState()
	changedState.Connection.Session.State = connection.Connecting
	changedState.Identities = []stateEvent.Identity{
		{
			Address:            "0xd535eba31e9bd2d7a4e34852e6292b359e5c77f7",
			RegistrationStatus: registry.RegisteredConsumer,
			ChannelAddress:     common.HexToAddress("0x000000000000000000000000000000000000000a"),
			Balance:            50,
			Earnings:           1,
			EarningsTotal:      100,
		},
	}
	h.ConsumeStateEvent(changedState)

	msg = <-results
	assert.Regexp(t, "^data:\\s?{.*}$", msg)
	msgJSON = strings.TrimPrefix(msg, "data: ")
	expectJSON = `
{
  "payload": {
    "nat_status": {
      "status": "",
      "error": ""
    },
    "service_info": null,
    "sessions": null,
    "consumer": {
      "connection": {
        "status": "Connecting"
      }
    },
    "identities": [
      {
        "id": "0xd535eba31e9bd2d7a4e34852e6292b359e5c77f7",
        "registration_status": "RegisteredConsumer",
        "channel_address": "0x000000000000000000000000000000000000000A",
        "balance": 50,
        "earnings": 1,
        "earnings_total": 100
      }
    ]
  },
  "type": "state-change"
}`
	assert.JSONEq(t, expectJSON, msgJSON)

	cancel()
	listener.Close()

	<-serveExit
}
