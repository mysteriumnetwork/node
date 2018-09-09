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

	"github.com/ethereum/go-ethereum/ethclient"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/payments/cli/helpers"
)

// Domain defines domain in which tequilAPI tequilapi_client will be executed
type Domain int

// Possible domain constants
const (
	Consumer Domain = 0
	Provider Domain = 1
)

// Flags to run a test
var (
	ehtRpcUrl = flag.String("geth.url", "http://localhost:8545", "RPC url of ethereum node")
)

// Provider flags
var (
	providerTequilapiHost = flag.String("provider.tequilapi-host", "localhost", "Specify Tequilapi host for provider")
	providerTequilapiPort = flag.Int("provider.tequilapi-port", 4050, "Specify Tequilapi port for provider")
)

// Consumer flags
var (
	consumerTequilapiHost = flag.String("consumer.tequilapi-host", "localhost", "Specify Tequilapi host for consumer")
	consumerTequilapiPort = flag.Int("consumer.tequilapi-port", 4050, "Specify Tequilapi port for consumer")
)

func newTequilapiClient(domain Domain) *tequilapi_client.Client {
	switch domain {
	case Consumer:
		return tequilapi_client.NewClient(*consumerTequilapiHost, *consumerTequilapiPort)
	case Provider:
		return tequilapi_client.NewClient(*providerTequilapiHost, *providerTequilapiPort)
	}
	return nil
}

func newEthClient() (*ethclient.Client, error) {
	ethClient, synced, err := helpers.LookupBackend(*ehtRpcUrl)
	if err != nil {
		return nil, err
	}
	<-synced //wait for sync to finish if any
	return ethClient, nil
}
