package bytescount_client

import (
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/server/dto"
)

type SessionStatsSender interface {
	send(bytesSent, bytesReceived int) error
}

type clientSessionStatsSender struct {
	mysteriumClient server.Client
	sessionId       string
}

func (sender *clientSessionStatsSender) send(bytesSent, bytesReceived int) error {
	return sender.mysteriumClient.SendSessionStats(sender.sessionId, dto.SessionStats{
		BytesSent:     bytesSent,
		BytesReceived: bytesReceived,
	})
}

func NewSessionStatsSender(mysteriumClient server.Client, sessionId string) SessionStatsSender {
	return &clientSessionStatsSender{mysteriumClient, sessionId}
}
