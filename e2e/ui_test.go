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

package e2e

import (
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	ssdp "github.com/koron/go-ssdp"
	"github.com/oleksandr/bonjour"
	"github.com/stretchr/testify/assert"
)

func TestLANDiscoverySSDP(t *testing.T) {
	services, err := ssdp.Search(ssdp.All, 1, "")
	assert.NoError(t, err)
	for _, service := range services {
		if service.Type == "upnp:rootdevice" && strings.Contains(service.Server, "UPnP/1.1 node/") {
			resp, err := http.Get(service.Location)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			return
		}
	}

	assert.Fail(t, "No SSDP service found")
}

func TestLANDiscoveryBonjour(t *testing.T) {
	resolver, err := bonjour.NewResolver(nil)
	if err != nil {
		log.Println("Failed to initialize resolver:", err.Error())
		os.Exit(1)
	}
	defer func() { resolver.Exit <- true }()

	results := make(chan *bonjour.ServiceEntry)
	go resolver.Browse("_mysterium-node._tcp", "", results)

	for {
		select {
		case <-time.After(3 * time.Second):
			assert.Fail(t, "No Bonjour service found")
			return
		case <-results:
			return
		}
	}
}

func TestBuiltinUIAccassable(t *testing.T) {
	resp, err := http.Get("http://" + *providerTequilapiHost + ":4449")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
