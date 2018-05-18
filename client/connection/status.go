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

package connection

import "github.com/mysterium/node/session"

// State represents list of possible connection states
type State string

const (
	// NotConnected means no connection exists
	NotConnected = State("NotConnected")
	// Connecting means that connection is started but not yet fully established
	Connecting = State("Connecting")
	// Connected means that fully established connection exists
	Connected = State("Connected")
	// Disconnecting means that connection close is in progress
	Disconnecting = State("Disconnecting")
	// Reconnecting means that connection is lost but underlying service is trying to reestablish it
	Reconnecting = State("Reconnecting")
)

// ConnectionStatus holds connection state and session id of the connnection
type ConnectionStatus struct {
	State     State
	SessionID session.SessionID
}

func statusConnecting() ConnectionStatus {
	return ConnectionStatus{Connecting, ""}
}

func statusConnected(sessionID session.SessionID) ConnectionStatus {
	return ConnectionStatus{Connected, sessionID}
}

func statusNotConnected() ConnectionStatus {
	return ConnectionStatus{NotConnected, ""}
}

func statusReconnecting() ConnectionStatus {
	return ConnectionStatus{Reconnecting, ""}
}

func statusDisconnecting() ConnectionStatus {
	return ConnectionStatus{Disconnecting, ""}
}
