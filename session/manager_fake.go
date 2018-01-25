package session

import "github.com/mysterium/node/identity"

//ManagerFake represents fake manager usually useful in tests
type ManagerFake struct{}

//Create function creates and returns fake session
func (manager *ManagerFake) Create() (Session, error) {
	return Session{"new-id", "new-config", identity.FromAddress("deadbeef")}, nil
}

//FindSession always returns empty session and signals that session is not found
func (manager *ManagerFake) FindSession(id SessionID) (Session, bool) {
	return Session{}, false
}
