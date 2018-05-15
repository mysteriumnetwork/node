package session

import (
	"github.com/mysterium/node/identity"
	"sync"
)

//ServiceConfigProvider interface defines configuration providing dependency
type ServiceConfigProvider interface {
	// ProvideServiceConfig is expected to provide serializable service configuration params from underlying service to remote party
	ProvideServiceConfig() (ServiceConfiguration, error)
}

// NewManager returns session manager which maintains a map of session id -> session
func NewManager(serviceConfigProvider ServiceConfigProvider, idGenerator Generator) *manager {
	return &manager{
		idGenerator:    idGenerator,
		configProvider: serviceConfigProvider,
		sessionMap:     make(map[SessionID]Session),
		creationLock:   sync.Mutex{},
	}
}

type manager struct {
	idGenerator    Generator
	configProvider ServiceConfigProvider
	sessionMap     map[SessionID]Session
	creationLock   sync.Mutex
}

func (manager *manager) Create(peerID identity.Identity) (sessionInstance Session, err error) {
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

func (manager *manager) FindSession(id SessionID) (Session, bool) {
	sessionInstance, found := manager.sessionMap[id]
	return sessionInstance, found
}
