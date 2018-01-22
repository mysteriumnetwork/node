package session

import "github.com/satori/go.uuid"

type Generator struct{}

func (generator *Generator) Generate() SessionID {
	return SessionID(uuid.NewV4().String())
}
