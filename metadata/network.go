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

package metadata

// NetworkDefinition structure holds all parameters which describe particular network
type NetworkDefinition struct {
	DiscoveryAPIAddress string
	BrokerAddress       string
}

// TestnetDefinition defines parameters for test network (currently default network)
var TestnetDefinition = NetworkDefinition{
	"https://testnet-api.mysterium.network/v1",
	"testnet-broker.mysterium.network",
}

// LocalnetDefinition defines parameters for local network (expects discovery and broker services on localhost)
var LocalnetDefinition = NetworkDefinition{
	"http://localhost:8001/v1",
	"localhost",
}

// DefaultNetwork defines default network values when no runtime parameters are given
var DefaultNetwork = TestnetDefinition
