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

package connection

import (
	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
)

type connInfo struct {
	fields      map[infoKey]string
	isConnected bool
}

type infoKey string

const (
	infIP          infoKey = "ip"
	infStatus      infoKey = "status"
	infLocation    infoKey = "location"
	infSessionID   infoKey = "sessionID"
	infProposal    infoKey = "proposal"
	infDuration    infoKey = "duration"
	infTransferred infoKey = "transferred"
	infThroughput  infoKey = "throughput"
	infSpent       infoKey = "spent"
	infIdentity    infoKey = "identity"
)

func newConnInfo() *connInfo {
	return &connInfo{
		fields:      make(map[infoKey]string),
		isConnected: false,
	}
}

func (i *connInfo) printAll() {
	i.printSingle("IP:", infIP)
	i.printSingle("Status:", infStatus)
	i.printSingle("Location:", infLocation)
	i.printSingle("Using Identity:", infIdentity)

	if !i.isConnected {
		return
	}

	i.printSingle("Session ID:", infSessionID)
	i.printSingle("Proposal:", infProposal)
	i.printSingle("Duration:", infDuration)
	i.printSingle("Transferred:", infTransferred)
	i.printSingle("Throughput:", infThroughput)
	i.printSingle("Spent:", infSpent)
}

func (i *connInfo) set(k infoKey, v string) {
	i.fields[k] = v
}

func (i *connInfo) printSingle(prefix string, k infoKey) {
	v, ok := i.fields[k]
	if !ok {
		clio.Warn(prefix, "No data retrieved")
		return
	}
	clio.Info(prefix, v)
}
