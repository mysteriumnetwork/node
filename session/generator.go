package session

import "github.com/satori/go.uuid"

type Generator struct{}

func (generator *Generator) Generate() SessionId {
	return SessionId(uuid.NewV4().String())
}
