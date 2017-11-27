package session

type Manager struct {
	Generator GeneratorInterface
	sessions  []SessionId
}

func (manager *Manager) Add(sid SessionId) {
	manager.sessions = append(manager.sessions, sid)
}

func (manager *Manager) Create() SessionId {
	id := manager.Generator.Generate()
	manager.Add(id)

	return id
}
