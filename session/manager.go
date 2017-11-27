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

func (manager *Manager) Get(sid SessionId) (sessionId SessionId) {
	for i := range manager.sessions {
		if manager.sessions[i] == sid {
			return manager.sessions[i]
		}
	}

	return
}
