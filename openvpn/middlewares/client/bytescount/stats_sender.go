package bytescount

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/server/dto"
	"github.com/mysterium/node/session"
)

type SessionStatsSender func(bytesSent, bytesReceived int) error

// NewSessionStatsSender returns new session stats handler, which sends statistics to server
func NewSessionStatsSender(mysteriumClient server.Client, sessionID session.SessionID, signer identity.Signer) SessionStatsHandler {
	sessionIDString := string(sessionID)
	return func(bytesSent, bytesReceived int) error {
		return mysteriumClient.SendSessionStats(
			sessionIDString,
			dto.SessionStats{
				BytesSent:     bytesSent,
				BytesReceived: bytesReceived,
			},
			signer,
		)
	}
}
