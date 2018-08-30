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

package e2e

import (
	"flag"

	"github.com/MysteriumNetwork/payments/cli/helpers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mysterium/node/tequilapi/client"
)

// Domain defines domain in which tequilAPI client will be executed
type Domain int

// Possible domain constants
const (
	Client Domain = 0
	Server Domain = 1
)

var tequilaClientHost = flag.String("tequila.client-host", "localhost", "Specify tequila client host for e2e tests")
var tequilaServiceHost = flag.String("tequila.service-host", "localhost", "Specify tequila service host for e2e tests")
var tequilaClientPort = flag.Int("tequila.client-port", 4050, "Specify tequila client port for e2e tests")
var tequilaServicePort = flag.Int("tequila.service-port", 4050, "Specify tequila service port for e2e tests")
var ethRPC = flag.String("geth.url", "http://localhost:8545", "Eth node RPC")

func newTequilaClient(domain Domain) *client.Client {
	switch domain {
	case Client:
		return client.NewClient(*tequilaClientHost, *tequilaClientPort)
	case Server:
		return client.NewClient(*tequilaServiceHost, *tequilaServicePort)
	}
	return nil
}

func newEthClient() (*ethclient.Client, error) {
	client, synced, err := helpers.LookupBackend(*ethRPC)
	if err != nil {
		return nil, err
	}
	<-synced //wait for sync to finish if any
	return client, nil
}
