package session

import "github.com/mysterium/node/session"

type Manager struct {
	Generator session.GeneratorInterface
	sessions  []session.SessionId
}

func (manager *Manager) Add(session session.Session) {
	manager.sessions = append(manager.sessions, session.Id)
}

func (manager *Manager) Create() session.Session {
	sessionInstance := session.Session{
		Id:     manager.Generator.Generate(),
		Config: "",
	}
	manager.Add(sessionInstance)

	return sessionInstance
}
