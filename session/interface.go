package session

type SessionID string

type Session struct {
	ID     SessionID
	Config string
}

type GeneratorInterface interface {
	Generate() SessionID
}

type ManagerInterface interface {
	Create() (Session, error)
	Add(Session)
}
