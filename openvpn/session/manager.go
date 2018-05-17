package session

import (
	"github.com/mysterium/node/identity"
	"sync"
	"github.com/mysterium/node/session"
)

// NewManager returns session manager which maintains a map of session id -> session
func NewManager(m session.Manager) *manager {
	return &manager{
		sessionManager:     m,
		sessionClientIDMap: make(map[session.SessionID]int),
		creationLock:   	sync.Mutex{},
	}
}

type manager struct {
	sessionManager session.Manager
	sessionClientIDMap     map[session.SessionID]int
	creationLock   sync.Mutex
}

// Create delegates session creation to underlying session.Manager
func (manager *manager) Create(peerID identity.Identity) (sessionInstance session.Session, err error) {
	return manager.sessionManager.Create(peerID)
}

// FindSession finds session and sets clientID if it is not set yet, returns false on clientID conflict
func (manager *manager) FindSession(clientID int, id session.SessionID) (session.Session, bool) {
	// start enumerating clients from '1', since non-existing key, might return '0' as clientID value
	clientID++
	sessionInstance, found := manager.sessionManager.FindSession(id)
	activeClientID := manager.sessionClientIDMap[id]
	if activeClientID == 0 {
		manager.sessionClientIDMap[id] = clientID
		return sessionInstance, found
	}

	if activeClientID != clientID {
		return session.Session{}, false
	}
	return sessionInstance, found
}
