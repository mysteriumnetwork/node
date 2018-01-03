package session

type ManagerFake struct{}

func (manager *ManagerFake) Create() (Session, error) {
	return Session{"new-id", "new-config"}, nil
}

func (manager *ManagerFake) Add(Session) {}
