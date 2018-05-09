package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
	"sync"
)

//ServiceConfigProvider interface defines configuration providing dependency
type ServiceConfigProvider interface {
	// ProvideServiceConfig function is expected to return service configuration which should be passed to service consumer
	ProvideServiceConfig() (session.VPNConfig, error)
}

// NewManager returns session manager which maintains a map of session id -> session
func NewManager(serviceConfigProvider ServiceConfigProvider, idGenerator session.Generator) *manager {
	return &manager{
		idGenerator:    idGenerator,
		configProvider: serviceConfigProvider,
		sessionMap:     make(map[session.SessionID]session.Session),
		creationLock:   sync.Mutex{},
	}
}

type manager struct {
	idGenerator    session.Generator
	configProvider ServiceConfigProvider
	sessionMap     map[session.SessionID]session.Session
	creationLock   sync.Mutex
}

func (manager *manager) Create(peerID identity.Identity) (sessionInstance session.Session, err error) {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()
	sessionInstance.ID = manager.idGenerator.Generate()
	sessionInstance.ConsumerID = peerID
	sessionInstance.Config, err = manager.configProvider.ProvideServiceConfig()
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
