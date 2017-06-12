package bytescount_client

import (
	"fmt"
	"github.com/MysteriumNetwork/node/openvpn"
	"github.com/MysteriumNetwork/node/server"
	"github.com/MysteriumNetwork/node/server/dto"
	"net"
	"regexp"
	"strconv"
	"time"
)

type middleware struct {
	mysteriumClient server.Client
	interval        time.Duration
	sessionId       string

	connection net.Conn
}

func NewMiddleware(mysteriumClient server.Client, sessionId string, interval time.Duration) openvpn.ManagementMiddleware {
	return &middleware{
		mysteriumClient: mysteriumClient,
		interval:        interval,
		sessionId:       sessionId,

		connection: nil,
	}
}

func (middleware *middleware) Start(connection net.Conn) error {
	middleware.connection = connection

	command := fmt.Sprintf("bytecount %d\n", int(middleware.interval.Seconds()))
	_, err := middleware.connection.Write([]byte(command))

	return err
}

func (middleware *middleware) Stop() error {
	_, err := middleware.connection.Write([]byte("bytecount 0\n"))
	return err
}

func (middleware *middleware) ConsumeLine(line string) (consumed bool, err error) {
	rule, err := regexp.Compile("^>BYTECOUNT:(.*),(.*)$")
	if err != nil {
		return
	}

	match := rule.FindStringSubmatch(line)
	consumed = len(match) > 0
	if !consumed {
		return
	}

	bytesIn, err := strconv.Atoi(match[1])
	if err != nil {
		return
	}

	bytesOut, err := strconv.Atoi(match[2])
	if err != nil {
		return
	}

	err = middleware.mysteriumClient.SessionSendStats(middleware.sessionId, dto.SessionStats{
		Id:            middleware.sessionId,
		BytesSent:     bytesOut,
		BytesReceived: bytesIn,
	})

	return
}
