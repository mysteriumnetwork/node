package bytescount

import (
	"net"
	"github.com/mysterium/node/openvpn"
	"regexp"
	"strconv"
)

type middleware struct {
	connection net.Conn
	bytesIn int
	bytesOut int
}

func NewMiddleware() openvpn.ManagementMiddleware {
	return &middleware{
		connection: nil,
		bytesIn: 0,
		bytesOut: 0,
	}
}

func (middleware *middleware) Start(connection net.Conn) error {
	middleware.connection = connection
	_, err := middleware.connection.Write([]byte("bytecount 5\n"))

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

	middleware.bytesIn += bytesIn
	middleware.bytesOut += bytesOut
	return
}