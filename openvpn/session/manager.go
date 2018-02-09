package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
	"sync"
)

// NewManager returns session manager which maintains a map of session id -> session
func NewManager(clientConfigGenerator openvpn.ClientConfigGenerator, idGenerator session.Generator) *manager {
	return &manager{
		idGenerator:          idGenerator,
		generateClientConfig: clientConfigGenerator,
		sessionMap:           make(map[session.SessionID]session.Session),
		creationLock:         sync.Mutex{},
	}
}

type manager struct {
	idGenerator          session.Generator
	generateClientConfig openvpn.ClientConfigGenerator
	sessionMap           map[session.SessionID]session.Session
	creationLock         sync.Mutex
}

func (manager *manager) Create(peerID identity.Identity) (sessionInstance session.Session, err error) {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()
	sessionInstance.ID = manager.idGenerator.Generate()
	sessionInstance.ConsumerID = peerID
	sessionInstance.Config, err = openvpn.ConfigToString(*manager.generateClientConfig().Config)
	if err != nil {
		return
	}

	manager.sessionMap[sessionInstance.ID] = sessionInstance
	return sessionInstance, nil
}

func (manager *manager) FindSession(id session.SessionID) (session.Session, bool) {
	sessionInstance, found := manager.sessionMap[id]
	return sessionInstance, found
}
