package bytescount_client

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/server/dto"
	"github.com/mysterium/node/session"
)

type SessionStatsSender func(bytesSent, bytesReceived int) error

func NewSessionStatsSender(mysteriumClient server.Client, sessionID session.SessionID, signer identity.Signer) SessionStatsSender {
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
