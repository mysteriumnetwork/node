/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package sse

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	nodeEvent "github.com/mysteriumnetwork/node/core/node/event"
	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/stretchr/testify/assert"
)

type mockStateProvider struct {
	stateToreturn stateEvent.State
}

func (msp *mockStateProvider) GetState() stateEvent.State {
	return msp.stateToreturn
}

func TestHandler_Stops(t *testing.T) {
	h := NewHandler(&mockStateProvider{})

	wait := make(chan struct{})
	go func() {
		h.serve()
		wait <- struct{}{}
	}()

	h.stop()
	<-wait
}

func TestHandler_ConsumeNodeEvent_Stops(t *testing.T) {
	h := NewHandler(&mockStateProvider{})
	me := nodeEvent.Payload{
		Status: nodeEvent.StatusStopped,
	}
	h.ConsumeNodeEvent(me)
	h.serve()
}

func TestHandler_ConsumeNodeEvent_Starts(t *testing.T) {
	h := NewHandler(&mockStateProvider{})
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
	h := NewHandler(msp)
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

	initialState := <-results
	assert.Equal(t, `data: {"payload":{"natStatus":{"status":"","error":""},"serviceInfo":null,"sessions":null},"type":"state-change"}`, initialState)

	changedState := msp.GetState()
	changedState.NATStatus = stateEvent.NATStatus{
		Status: "mass panic",
		Error:  "cookie prices rise drastically",
	}
	h.ConsumeStateEvent(changedState)

	newState := <-results
	assert.Equal(t, `data: {"payload":{"natStatus":{"status":"mass panic","error":"cookie prices rise drastically"},"serviceInfo":null,"sessions":null},"type":"state-change"}`, newState)
	cancel()
	listener.Close()

	<-serveExit
}
