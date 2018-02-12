package auth

import (
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/management"
	"regexp"
)

// CredentialsProvider returns client's current auth primitives (i.e. customer identity signature / node's sessionId)
type CredentialsProvider func() (username string, password string, err error)

type middleware struct {
	fetchCredentials CredentialsProvider
	commandWriter    management.Connection
	lastUsername     string
	lastPassword     string
	state            openvpn.State
}

// NewMiddleware creates client user_auth challenge authentication middleware
func NewMiddleware(credentials CredentialsProvider) *middleware {
	return &middleware{
		fetchCredentials: credentials,
		commandWriter:    nil,
	}
}

func (m *middleware) Start(commandWriter management.Connection) error {
	m.commandWriter = commandWriter
	log.Info("starting client user-pass provider middleware")
	return nil
}

func (m *middleware) Stop(commandWriter management.Connection) error {
	return nil
}

func (m *middleware) ConsumeLine(line string) (consumed bool, err error) {
	rule, err := regexp.Compile("^>PASSWORD:Need 'Auth' username/password$")
	if err != nil {
		return false, err
	}

	match := rule.FindStringSubmatch(line)
	if len(match) == 0 {
		return false, nil
	}
	username, password, err := m.fetchCredentials()
	log.Info("authenticating user ", username, " with pass: ", password)

	err = m.commandWriter.PrintfLine("password 'Auth' %s", password)
	if err != nil {
		return true, err
	}

	err = m.commandWriter.PrintfLine("username 'Auth' %s", username)
	if err != nil {
		return true, err
	}
	return true, nil
}
