package state

import (
	"errors"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/management"
	"regexp"
	"strings"
)

// Callback is called when openvpn process state changes
type Callback func(state openvpn.State)

const stateEventMatcher = "^STATE:\\d+,([a-zA-Z]+),.*$"

var errLineMismatch = errors.New("line didn't match")

type middleware struct {
	listeners []Callback
}

func NewMiddleware(listeners ...Callback) management.Middleware {
	return &middleware{
		listeners: listeners,
	}
}

func (middleware *middleware) Start(commandWriter management.Connection) error {
	_, lines, err := commandWriter.MultiOutputCommand("line on all")
	if err != nil {
		return err
	}
	for _, line := range lines {
		state, err := extractOpenvpnState(line)
		if err != nil {
			return err
		}
		middleware.callListeners(state)
	}
	return nil
}

func (middleware *middleware) Stop(commandWriter management.Connection) error {
	_, err := commandWriter.SingleOutputCommand("state off")
	return err
}

func (middleware *middleware) ConsumeLine(line string) (bool, error) {
	state, err := extractOpenvpnState(strings.TrimPrefix(line, ">"))
	if err != nil {
		switch err {
		case errLineMismatch:
			return false, nil
		default:
			return true, err
		}
	}

	middleware.callListeners(state)
	return true, nil
}

func (middleware *middleware) Subscribe(listener Callback) {
	middleware.listeners = append(middleware.listeners, listener)
}

func (middleware *middleware) callListeners(state openvpn.State) {
	for _, listener := range middleware.listeners {
		listener(state)
	}
}

func extractOpenvpnState(line string) (openvpn.State, error) {
	rule, err := regexp.Compile(stateEventMatcher)
	if err != nil {
		return openvpn.STATE_UNDEFINED, err
	}

	match := rule.FindStringSubmatch(line)
	if len(match) < 2 {
		return openvpn.STATE_UNDEFINED, errLineMismatch
	}

	return openvpn.State(match[1]), nil
}
