package session

import (
	"github.com/mysterium/node/identity"
	"sync"
	"github.com/mysterium/node/session"
)


// NewManager returns session manager which maintains a map of session id -> session
func NewManager(serviceConfigProvider session.ServiceConfigProvider, idGenerator session.Generator) *manager {
	return &manager{
		sessionManager:     session.NewManager(serviceConfigProvider, idGenerator),
		sessionClientIDMap: make(map[session.SessionID]int),
		creationLock:   	sync.Mutex{},
	}
}

type manager struct {
	sessionManager session.Manager
	sessionClientIDMap     map[session.SessionID]int
	creationLock   sync.Mutex
}

func (manager *manager) Create(peerID identity.Identity) (sessionInstance session.Session, err error) {
	return manager.sessionManager.Create(peerID)
}

func (manager *manager) FindSession(id session.SessionID) (session.Session, bool) {
	return manager.sessionManager.FindSession(id)
}

// FindUpdateSessionWithClientID finds session and sets clientID if it is not set yet, returns false on clientID conflict
func (manager *manager) FindUpdateSessionWithClientID(clientID int, id session.SessionID) (session.Session, bool) {
	sessionInstance, found := manager.FindSession(id)
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
