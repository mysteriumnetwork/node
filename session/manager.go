package session

type Manager struct {
	Sessions []SessionId
}

func (manager *Manager) Create() (SessionId, error) {
	sid, err := GenerateSessionId()
	if err != nil {
		return sid, err
	}

	for !manager.sessionIsUnique(sid) {
		sid, err := GenerateSessionId()
		if err != nil {
			return sid, err
		}
	}

	manager.Sessions = append(manager.Sessions, sid)

	return sid, err
}

func (manager *Manager) sessionIsUnique(sid SessionId) bool {
	for _, v := range manager.Sessions {
		if v == sid {
			return false
		}
	}

	return true
}
