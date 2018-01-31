package session

import "github.com/mysterium/node/identity"

//SessionID represents session id type
type SessionID string

//Session structure holds all required information about current session between service consumer and provider
type Session struct {
	ID         SessionID
	Config     string
	ConsumerID identity.Identity
}

//Generator defines method for session id generation
type Generator interface {
	Generate() SessionID
}

//Manager defines methods for session management
type Manager interface {
	Create(identity.Identity) (Session, error)
	FindSession(SessionID) (Session, bool)
}
