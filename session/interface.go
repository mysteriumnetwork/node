package session

import "github.com/mysterium/node/identity"

//SessionID represents session id type
type SessionID string

//Session structure holds all required information about current session between service consumer and provider
type Session struct {
	ID           SessionID
	Config       string
	PeerIdentity identity.Identity
}

//GeneratorInterface defines method for session id generation
type GeneratorInterface interface {
	Generate() SessionID
}

//ManagerInterface defines methods for session management
type ManagerInterface interface {
	Create() (Session, error)
	FindSession(SessionID) (Session, bool)
}
