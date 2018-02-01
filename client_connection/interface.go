package client_connection

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
)

type State string

const (
	NotConnected  = State("NotConnected")
	Connecting    = State("Connecting")
	Connected     = State("Connected")
	Disconnecting = State("Disconnecting")
)

type ConnectionStatus struct {
	State     State
	SessionID session.SessionID
	LastError error
}

type Manager interface {
	Connect(consumerID identity.Identity, providerID identity.Identity) error
	Status() ConnectionStatus
	Disconnect() error
}
