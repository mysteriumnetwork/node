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
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/ui/versionmanager"

	"github.com/mysteriumnetwork/node/requests"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

type jwtAuth struct {
}

func (j *jwtAuth) ValidateToken(token string) (bool, error) {
	return false, nil
}

func Test_Server_ServesHTML(t *testing.T) {
	// given
	tmpDIr, err := os.MkdirTemp("", "nodeuiversiontest")
	assert.NoError(t, err)
	defer os.Remove(tmpDIr)

	config, err := versionmanager.NewVersionConfig("/tmp")
	assert.NoError(t, err)

	s := NewServer(
		"localhost",
		55565,
		"localhost",
		55564,
		&jwtAuth{},
		requests.NewHTTPClient("0.0.0.0", requests.DefaultTimeout),
		config,
	)
	s.discovery = &mockDiscovery{}
	s.Serve()
	time.Sleep(time.Millisecond * 100)

	// when
	nilResp, err := http.Get("http://:55565/")
	assert.Nil(t, err)
	defer nilResp.Body.Close()

	// then
	root, err := html.Parse(nilResp.Body)
	assert.Nil(t, err)
	body := hasBody(root)
	assert.True(t, body)

	s.Stop()
}

func hasBody(n *html.Node) bool {
	fs := n.FirstChild
	if fs == nil {
		return false
	}
	if fs.Type == html.ElementNode && n.Data == "body" {
		return true
	}

	return hasBody(fs.NextSibling)
}

type mockDiscovery struct{}

func (md *mockDiscovery) Start() error { return nil }
func (md *mockDiscovery) Stop() error  { return nil }
