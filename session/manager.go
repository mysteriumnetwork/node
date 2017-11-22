package session

type Manager struct {
	Sessions []SessionId
}

func (manager *Manager) Create() (sid SessionId) {
	sid = GenerateSessionId()

	manager.Sessions = append(manager.Sessions, sid)

	return sid
}