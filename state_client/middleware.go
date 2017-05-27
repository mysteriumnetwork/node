package state_client

import (
	"fmt"
	"github.com/mysterium/node/openvpn"
	"net"
	"regexp"
)

type middleware struct {
	connection net.Conn
}

type clientState string

const STATE_CONNECTING = clientState("CONNECTING")
const STATE_WAIT = clientState("WAIT")
const STATE_AUTH = clientState("AUTH")
const STATE_GET_CONFIG = clientState("GET_CONFIG")
const STATE_ASSIGN_IP = clientState("ASSIGN_IP")
const STATE_ADD_ROUTES = clientState("ADD_ROUTES")
const STATE_CONNECTED = clientState("CONNECTED")
const STATE_RECONNECTING = clientState("RECONNECTING")
const STATE_EXITING = clientState("EXITING")

func NewMiddleware() openvpn.ManagementMiddleware {
	return &middleware{
		connection: nil,
	}
}

func (middleware *middleware) Start(connection net.Conn) error {
	middleware.connection = connection

	_, err := middleware.connection.Write([]byte("state on\n"))
	return err
}

func (middleware *middleware) Stop() error {
	_, err := middleware.connection.Write([]byte("state off\n"))
	return err
}

func (middleware *middleware) ConsumeLine(line string) (consumed bool, err error) {
	rule, err := regexp.Compile("^>STATE:(.*)$")
	if err != nil {
		return
	}

	match := rule.FindStringSubmatch(line)
	consumed = len(match) > 0
	if !consumed {
		return
	}

	state := match[1]
	fmt.Println("State ISSSSSSSS: ", state)

	return
}
