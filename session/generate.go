package session

import (
	"github.com/satori/go.uuid"
)

type SessionId string

func GenerateSessionId() (sid SessionId) {
	sid = SessionId(uuid.NewV4().String())

	return
}
