package bytescount

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/server/dto"
	"github.com/mysterium/node/session"
)

// SessionStatsSender sends statistics to server
type SessionStatsSender func(bytesSent, bytesReceived int) error

// NewSessionStatsSender returns new session stats handler, which sends statistics to server
func NewSessionStatsSender(mysteriumClient server.Client, sessionID session.SessionID, providerID identity.Identity, signer identity.Signer, ConsumerCountry string) SessionStatsHandler {
	sessionIDString := string(sessionID)
	return func(sessionStats SessionStats) error {
		return mysteriumClient.SendSessionStats(
			sessionIDString,
			dto.SessionStats{
				BytesSent:       sessionStats.BytesSent,
				BytesReceived:   sessionStats.BytesReceived,
				ProviderID:      providerID.Address,
				ConsumerCountry: ConsumerCountry,
			},
			signer,
		)
	}
}
