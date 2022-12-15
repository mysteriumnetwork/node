/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package bandwidth

import (
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/datasize"
)

// AppTopicConnectionThroughput represents the session throughput topic.
const AppTopicConnectionThroughput = "Throughput"

// AppEventConnectionThroughput represents a session throughput event.
type AppEventConnectionThroughput struct {
	UUID        string
	Throughput  Throughput
	SessionInfo connectionstate.Status
}

// Throughput represents the current(moment) download and upload speeds.
type Throughput struct {
	Up, Down datasize.BitSpeed
}
