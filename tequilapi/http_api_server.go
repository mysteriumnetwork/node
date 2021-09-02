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
	"net"
	"net/http"
	"strings"

	"github.com/mysteriumnetwork/node/core/node"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var corsConfig = cors.Config{
	AllowMethods: []string{
		"GET",
		"HEAD",
		"POST",
		"PUT",
		"DELETE",
		"CONNECT",
		"OPTIONS",
		"TRACE",
		"PATCH",
	},
	AllowHeaders: []string{
		"Origin",
		"Content-Length",
		"Content-Type",
		"Cache-Control",
		"X-XSRF-TOKEN",
		"X-CSRF-TOKEN",
	},
	AllowCredentials: true,
	AllowOriginFunc: func(_ string) bool {
		return true
	},
}

// APIServer interface represents control methods for underlying http api server
type APIServer interface {
	Wait() error
	StartServing()
	Stop()
	Address() (string, error)
}

type apiServer struct {
	errorChannel chan error
	listener     net.Listener

	gin *gin.Engine
}

// NewServer creates http api server for given address port and http handler
func NewServer(
	listener net.Listener,
	nodeOptions node.Options,
	handlers []func(e *gin.Engine) error,
) (APIServer, error) {
	gin.SetMode(modeFromOptions(nodeOptions))
	g := gin.New()
	g.Use(gin.Recovery())
	g.Use(cors.New(corsConfig))

	for _, h := range handlers {
		err := h(g)
		if err != nil {
			return nil, err
		}
	}

	server := apiServer{
		errorChannel: make(chan error, 1),
		listener:     listener,

		gin: g,
	}

	return &server, nil
}

func modeFromOptions(options node.Options) string {
	if options.FlagTequilapiDebugMode {
		return gin.DebugMode
	}

	return gin.ReleaseMode
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
	go server.serve()
	address, err := server.Address()
	if err != nil {
		log.Error().Err(err).Msg("Could not get tequila server address")
		return
	}
	log.Info().Msgf("API started on: %s", address)
}

func (server *apiServer) serve() {
	server.errorChannel <- http.Serve(server.listener, server.gin)
}

func extractBoundAddress(listener net.Listener) (string, error) {
	addr := listener.Addr()
	parts := strings.Split(addr.String(), ":")
	if len(parts) < 2 {
		return "", errors.New("Unable to locate address: " + addr.String())
	}
	return addr.String(), nil
}
