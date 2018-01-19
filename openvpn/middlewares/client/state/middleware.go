package state

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares"
	"net"
	"regexp"
)

type middleware struct {
	listeners  []clientStateCallback
	connection net.Conn
	state      middlewares.State
}

type clientStateCallback func(state middlewares.State) error

func NewMiddleware(listeners ...clientStateCallback) openvpn.ManagementMiddleware {
	return &middleware{
		listeners:  listeners,
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

func (middleware *middleware) State() middlewares.State {
	return middleware.state
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

	state := middlewares.State(match[1])
	for _, listener := range middleware.listeners {
		err = listener(state)
		if err != nil {
			return
		}
	}

	return
}

func (middleware *middleware) Subscribe(listener clientStateCallback) {
	middleware.listeners = append(middleware.listeners, listener)
}
