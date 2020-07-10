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

	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
)

// Provider flags
var (
	providerTequilapiHost = flag.String("provider.tequilapi-host", "localhost", "Specify Tequilapi host for provider")
	providerTequilapiPort = flag.Int("provider.tequilapi-port", 4050, "Specify Tequilapi port for provider")
)

// Consumer flags
var (
	consumerTequilapiPort = flag.Int("consumer.tequilapi-port", 4050, "Specify Tequilapi port for consumer")
	consumerServices      = flag.String("consumer.services", "openvpn,noop,wireguard", "Comma separated list of services to try and use")
)

func newTequilapiConsumer(host string) *tequilapi_client.Client {
	return tequilapi_client.NewClient(host, *consumerTequilapiPort)
}

func newTequilapiProvider() *tequilapi_client.Client {
	return tequilapi_client.NewClient(*providerTequilapiHost, *providerTequilapiPort)
}
