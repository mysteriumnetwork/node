package client_connection

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
)

type State string

const (
	NotConnected = State("NotConnected")
	Connecting   = State("Connecting")
	Connected    = State("Connected")
)

type ConnectionStatus struct {
	State     State
	SessionId session.SessionId
	LastError error
}

type Manager interface {
	Connect(identity identity.Identity, NodeKey string) error
	Status() ConnectionStatus
	Disconnect() error
	Wait() error
}
