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

package ui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	godvpnweb "github.com/mysteriumnetwork/go-dvpn-web"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/ui/discovery"
	"github.com/rs/zerolog/log"
)

const tequilapiUrlPrefix = "/tequilapi"

// Server represents our web UI server
type Server struct {
	srvs      []*http.Server
	discovery discovery.LANDiscovery
}

type jwtAuthenticator interface {
	ValidateToken(token string) (bool, error)
}

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
	AllowOriginFunc: func(origin string) bool {
		return true
	},
}

// NewServer creates a new instance of the server for the given port
// you can chain addresses with ',' i.e. "192.168.0.1,127.0.0.1"
func NewServer(bindAddress string, port int, tequilapiAddress string, tequilapiPort int, authenticator jwtAuthenticator, httpClient *requests.HTTPClient) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.NoRoute(ReverseTequilapiProxy(tequilapiAddress, tequilapiPort, authenticator))
	r.Use(cors.New(corsConfig))

	r.StaticFS("/", godvpnweb.Assets)

	addrs := strings.Split(bindAddress, ",")

	var srvs []*http.Server
	for _, addr := range addrs {
		s := &http.Server{
			Addr:    fmt.Sprintf("%v:%v", addr, port),
			Handler: r,
		}
		srvs = append(srvs, s)
	}

	return &Server{
		srvs:      srvs,
		discovery: discovery.NewLANDiscoveryService(port, httpClient),
	}
}

// Serve starts servers
func (s *Server) Serve() {
	go func() {
		err := s.discovery.Start()
		if err != nil {
			log.Error().Err(err).Msg("Failed to start local discovery service")
		}
	}()

	for _, srv := range s.srvs {
		go startListen(srv)
	}
}

func startListen(s *http.Server) {
	log.Info().Msgf("UI starting on: %s", s.Addr)
	err := s.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Err(err).Msg("UI server crashed")
	}
}

// Stop stops servers
func (s *Server) Stop() {
	err := s.discovery.Stop()
	if err != nil {
		log.Error().Err(err).Msg("Failed to stop local discovery service")
	}

	// give the server a few seconds to shut down properly in case a request is waiting somewhere
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for _, srv := range s.srvs {
		err = srv.Shutdown(ctx)
		log.Info().Err(err).Msg("Server stopped")
	}
}
