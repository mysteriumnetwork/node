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
	"github.com/mysterium/node/tequilapi/client"
)

var host = flag.String("tequila.host", "localhost", "Specify tequila host for e2e tests")
var port = flag.Int("tequila.port", 4050, "Specify tequila port for e2e tests")

func newTequilaClient() *client.Client {
	return client.NewClient(*host, *port)
}
