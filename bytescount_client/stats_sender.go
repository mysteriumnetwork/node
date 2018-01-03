package bytescount_client

import (
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/server/dto"
	"github.com/mysterium/node/session"
)

type SessionStatsSender func(bytesSent, bytesReceived int) error

func NewSessionStatsSender(mysteriumClient server.Client, sessionId session.SessionId) SessionStatsSender {
	sessionIdString := string(sessionId)
	return SessionStatsSender(func(bytesSent, bytesReceived int) error {
		return mysteriumClient.SendSessionStats(sessionIdString, dto.SessionStats{
			BytesSent:     bytesSent,
			BytesReceived: bytesReceived,
		})
	})
}
