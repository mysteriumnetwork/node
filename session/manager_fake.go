package session

import "github.com/mysterium/node/identity"

// ManagerFake represents fake manager usually useful in tests
type ManagerFake struct{}

var fakeConfig = struct {
	Param1 string
	Param2 int
}{
	"string-param",
	123,
}

// Create function creates and returns fake session
func (manager *ManagerFake) Create(peerID identity.Identity) (Session, error) {
	return Session{"new-id", fakeConfig, peerID}, nil
}

// FindSession always returns empty session and signals that session is not found
func (manager *ManagerFake) FindSession(id SessionID) (Session, bool) {
	return Session{}, false
}

// RemoveSession removes given session from underlying session manager
func (manager *ManagerFake) RemoveSession(id SessionID) {
	// just stub
}
