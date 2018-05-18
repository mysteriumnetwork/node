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

package discovery

// TypeContactNATSV1 defines V1 format for NATS contact
const TypeContactNATSV1 = "nats/v1"

// ContactNATSV1 is definition of NATS contact
type ContactNATSV1 struct {
	// Topic on which client is getting message
	Topic string `json:"topic"`
	// NATS servers used by node and should be contacted via
	BrokerAddresses []string `json:"broker_addresses"`
}
