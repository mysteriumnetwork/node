package session

type SessionId string

type Session struct {
	Id     SessionId
	Config string
}

type GeneratorInterface interface {
	Generate() SessionId
}

type ManagerInterface interface {
	Create() Session
	Add(Session)
}
