package state

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/management"
	"regexp"
)

// Callback is called when openvpn process state changes
type Callback func(state openvpn.State)

type middleware struct {
	listeners []Callback
}

func NewMiddleware(listeners ...Callback) management.Middleware {
	return &middleware{
		listeners: listeners,
	}
}

func (middleware *middleware) Start(commandWriter management.Connection) error {
	return commandWriter.PrintfLine("state on all")
}

func (middleware *middleware) Stop(commandWriter management.Connection) error {
	return commandWriter.PrintfLine("state off")
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

func (middleware *middleware) Subscribe(listener Callback) {
	middleware.listeners = append(middleware.listeners, listener)
}
