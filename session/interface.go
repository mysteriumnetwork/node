package session

type SessionId string

type GeneratorInterface interface {
	Generate() SessionId
}

type ManagerInterface interface {
	Create() SessionId
	Add(SessionId)
}
