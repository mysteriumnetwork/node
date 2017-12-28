package session

import "github.com/mysterium/node/session"

func NewManager() *manager {
	return &manager{
		generator: &session.Generator{},
		sessions:  make([]session.SessionId, 0),
	}
}

type manager struct {
	generator session.GeneratorInterface
	sessions  []session.SessionId
}

func (manager *manager) Add(session session.Session) {
	manager.sessions = append(manager.sessions, session.Id)
}

func (manager *manager) Create() session.Session {
	sessionInstance := session.Session{
		Id:     manager.generator.Generate(),
		Config: "",
	}
	manager.Add(sessionInstance)

	return sessionInstance
}
