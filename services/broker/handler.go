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
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/jcuga/golongpoll"
	"github.com/rs/zerolog/log"
)

type handler struct {
	lp *golongpoll.LongpollManager
}

func newP2PHandler() *handler {
	manager, err := golongpoll.StartLongpoll(golongpoll.Options{
		LoggingEnabled: true,
	})
	if err != nil {
		panic(err)
	}

	return &handler{
		lp: manager,
	}
}

func (h *handler) brokerMsgHandle(w http.ResponseWriter, req *http.Request) {
	log.Info().Msgf("Received broker message for: %s", req.URL.RequestURI())
	id := strings.TrimPrefix(req.URL.Path, "/msg/")

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
		h.lp.Publish(id+"_init", msg)

		q := req.URL.Query()
		q.Add("timeout", "120")
		q.Add("category", id+"_push")
		req.URL.RawQuery = q.Encode()

		h.lp.SubscriptionHandler(w, req)

	case "push":
		log.Info().Msgf("Received broker message for push: %s", req.URL.Path)
		h.lp.Publish(id+"_push", msg)

	case "ack":
		log.Info().Msgf("Received broker message for ack: %s", req.URL.Path)
		h.lp.Publish(id+"_ack", msg)
	}
}

func (h *handler) brokerPollInitHandle(w http.ResponseWriter, req *http.Request) {
	id := strings.TrimPrefix(req.URL.Path, "/poll/init/")
	q := req.URL.Query()
	q.Add("timeout", "120")
	q.Add("category", id+"_init")
	req.URL.RawQuery = q.Encode()

	h.lp.SubscriptionHandler(w, req)
}

func (h *handler) brokerPollAckHandle(w http.ResponseWriter, req *http.Request) {
	id := strings.TrimPrefix(req.URL.Path, "/poll/ack/")
	q := req.URL.Query()
	q.Add("timeout", "120")
	q.Add("category", id+"_ack")
	req.URL.RawQuery = q.Encode()

	h.lp.SubscriptionHandler(w, req)
}
