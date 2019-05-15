/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package tequilapi

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

// APIServer interface represents control methods for underlying http api server
type APIServer interface {
	Wait() error
	StartServing()
	Stop()
	Address() (string, error)
}

type apiServer struct {
	errorChannel chan error
	handler      http.Handler
	listener     net.Listener
}

// NewServer creates http api server for given address port and http handler
func NewServer(listener net.Listener, handler http.Handler, corsPolicy CorsPolicy) APIServer {
	server := apiServer{
		errorChannel: make(chan error, 1),
		handler:      DisableCaching(ApplyCors(handler, corsPolicy)),
		listener:     listener,
	}
	return &server
}

// Stop method stops underlying http server
func (server *apiServer) Stop() {
	server.listener.Close()
}

// Wait method waits for http server to finish handling requests (i.e. when Stop() was called)
func (server *apiServer) Wait() error {
	return <-server.errorChannel
}

// Address method returns bind port for given http server (useful when random port is used)
func (server *apiServer) Address() (string, error) {
	return extractBoundAddress(server.listener)
}

// StartServing starts http request serving
func (server *apiServer) StartServing() {
	go server.serve(server.handler)
}

func (server *apiServer) serve(handler http.Handler) {
	server.errorChannel <- http.Serve(server.listener, handler)
}

func extractBoundAddress(listener net.Listener) (string, error) {
	addr := listener.Addr()
	parts := strings.Split(addr.String(), ":")
	if len(parts) < 2 {
		return "", errors.New("Unable to locate address: " + addr.String())
	}
	return addr.String(), nil
}
