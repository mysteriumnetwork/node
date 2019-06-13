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
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/koron/go-ssdp"
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
	assert.NoError(t, err)
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

func TestBuiltinUIAccessible(t *testing.T) {
	req, err := http.NewRequest("GET", "http://"+*providerTequilapiHost+":4449", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("myst:mystberry")))
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
