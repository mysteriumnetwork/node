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
	LastError error
}

func statusError(err error) ConnectionStatus {
	return ConnectionStatus{NotConnected, "", err}
}

func statusConnecting() ConnectionStatus {
	return ConnectionStatus{Connecting, "", nil}
}

func statusConnected(sessionID session.SessionID) ConnectionStatus {
	return ConnectionStatus{Connected, sessionID, nil}
}

func statusNotConnected() ConnectionStatus {
	return ConnectionStatus{NotConnected, "", nil}
}

func statusReconnecting() ConnectionStatus {
	return ConnectionStatus{Reconnecting, "", nil}
}

func statusDisconnecting() ConnectionStatus {
	return ConnectionStatus{Disconnecting, "", nil}
}
