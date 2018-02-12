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
