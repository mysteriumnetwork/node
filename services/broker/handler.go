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

package broker

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/jcuga/golongpoll"
	"github.com/rs/zerolog/log"
)

type handler struct {
	lp *golongpoll.LongpollManager
	id string

	mu     sync.Mutex
	rx, tx uint64
}

type readCloserCounter struct {
	rc    io.ReadCloser
	count func(n int)
}

func (r *readCloserCounter) Read(p []byte) (n int, err error) {
	n, err = r.rc.Read(p)
	r.count(n)

	return n, err
}

func (r *readCloserCounter) Close() error {
	return r.rc.Close()
}

func newBrokerHandler(id string) (*handler, error) {
	manager, err := golongpoll.StartLongpoll(golongpoll.Options{
		LoggingEnabled: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start longpoll")
	}

	return &handler{
		lp: manager,
		id: id,
	}, nil
}

func (h *handler) countRx(rc io.ReadCloser) *readCloserCounter {
	return &readCloserCounter{
		rc: rc,
		count: func(n int) {
			h.mu.Lock()
			defer h.mu.Unlock()

			h.rx += uint64(n)
		},
	}
}

func (h *handler) brokerHandle(w http.ResponseWriter, req *http.Request) {
	path := strings.SplitN(req.URL.Path, "/", 5)
	if len(path) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := path[1]
	serviceType := path[2]
	t := path[3]

	if id != h.id {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	req.Body = h.countRx(req.Body)

	switch t {
	case "msg":
		h.brokerMsgHandle(serviceType, w, req)
	case "init":
		h.brokerInitHandle(serviceType, w, req)
	case "ack":
		h.brokerAckHandle(serviceType, w, req)
	}
}

func (h *handler) brokerMsgHandle(serviceType string, w http.ResponseWriter, req *http.Request) {
	log.Info().Msgf("Received broker message for: %s", req.URL.RequestURI())

	t, ok := req.URL.Query()["type"]
	if !ok || len(t[0]) < 1 {
		log.Error().Msgf("Argument 'type' missing: %s", req.URL.RequestURI())
		return
	}

	msg, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}

	switch t[0] {
	case "init":
		log.Info().Msgf("Received broker message for init: %s", req.URL.Path)
		h.lp.Publish(h.id+serviceType+"_init", msg)

		q := req.URL.Query()
		q.Add("timeout", "120")
		q.Add("category", h.id+serviceType+"_push")
		req.URL.RawQuery = q.Encode()

		h.lp.SubscriptionHandler(w, req)

	case "push":
		log.Info().Msgf("Received broker message for push: %s", req.URL.Path)
		h.lp.Publish(h.id+serviceType+"_push", msg)

	case "ack":
		log.Info().Msgf("Received broker message for ack: %s", req.URL.Path)
		h.lp.Publish(h.id+serviceType+"_ack", msg)
	}
}

func (h *handler) brokerInitHandle(serviceType string, w http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	q.Add("timeout", "120")
	q.Add("category", h.id+serviceType+"_init")
	req.URL.RawQuery = q.Encode()

	h.lp.SubscriptionHandler(w, req)
}

func (h *handler) brokerAckHandle(serviceType string, w http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	q.Add("timeout", "120")
	q.Add("category", h.id+serviceType+"_ack")
	req.URL.RawQuery = q.Encode()

	h.lp.SubscriptionHandler(w, req)
}

func (h *handler) stats() (rx, tx uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.rx, h.tx
}
