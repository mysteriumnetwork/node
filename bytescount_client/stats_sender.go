package bytescount_client

import (
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/server/dto"
)

type SessionStatsSender interface {
	send(sessionId string, bytesSent, bytesReceived int) error
}

type clientSessionStatsSender struct {
	mysteriumClient server.Client
}

func (sender *clientSessionStatsSender) send(sessionId string, bytesSent, bytesReceived int) error {
	return sender.mysteriumClient.SendSessionStats(sessionId, dto.SessionStats{
		BytesSent:     bytesSent,
		BytesReceived: bytesReceived,
	})
}

func NewSessionStatsSender(mysteriumClient server.Client) SessionStatsSender {
	return &clientSessionStatsSender{mysteriumClient}
}
