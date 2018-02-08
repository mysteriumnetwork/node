package state

import (
	"github.com/mysterium/node/openvpn"
	"net"
	"regexp"
)

type middleware struct {
	listeners  []ClientStateCallback
	connection net.Conn
}

type ClientStateCallback func(state openvpn.State)

func NewMiddleware(listeners ...ClientStateCallback) openvpn.ManagementMiddleware {
	return &middleware{
		listeners:  listeners,
		connection: nil,
	}
}

func (middleware *middleware) Start(connection net.Conn) error {
	middleware.connection = connection

	_, err := middleware.connection.Write([]byte("state on all\n"))
	return err
}

func (middleware *middleware) Stop() error {
	_, err := middleware.connection.Write([]byte("state off\n"))
	return err
}

func (middleware *middleware) ConsumeLine(line string) (consumed bool, err error) {
	rule, err := regexp.Compile("^>STATE:\\d+,([a-zA-Z]+),.*$")
	if err != nil {
		return
	}

	match := rule.FindStringSubmatch(line)
	consumed = len(match) > 0
	if !consumed {
		return
	}

	state := openvpn.State(match[1])
	for _, listener := range middleware.listeners {
		listener(state)
	}

	return
}

func (middleware *middleware) Subscribe(listener ClientStateCallback) {
	middleware.listeners = append(middleware.listeners, listener)
}
