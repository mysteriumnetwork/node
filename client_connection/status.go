package client_connection

import "github.com/mysterium/node/session"

type State string

const (
	NotConnected  = State("NotConnected")
	Connecting    = State("Connecting")
	Connected     = State("Connected")
	Disconnecting = State("Disconnecting")
	Reconnecting  = State("Reconnecting")
)

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
