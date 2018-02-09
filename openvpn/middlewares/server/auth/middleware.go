package auth

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/openvpn"
	"regexp"
	"strconv"
)

// CredentialsChecker callback checks given auth primitives (i.e. customer identity signature / node's sessionId)
type CredentialsChecker func(username, password string) (bool, error)

type middleware struct {
	checkCredentials CredentialsChecker
	commandWriter    openvpn.CommandWriter
	lastUsername     string
	lastPassword     string
	clientID         int
	keyID            int
}

// NewMiddleware creates server user_auth challenge authentication middleware
func NewMiddleware(credentialsChecker CredentialsChecker) *middleware {
	return &middleware{
		checkCredentials: credentialsChecker,
		commandWriter:    nil,
	}
}

func (m *middleware) Start(commandWriter openvpn.CommandWriter) error {
	m.commandWriter = commandWriter
	return nil
}

func (m *middleware) Stop(commandWriter openvpn.CommandWriter) error {
	return nil
}

func (m *middleware) checkReAuth(line string) (cont bool, consumed bool, err error) {

	rule, err := regexp.Compile("^>CLIENT:REAUTH,(\\d),(\\d)$")
	if err != nil {
		return false, false, err
	}

	match := rule.FindStringSubmatch(line)
	if len(match) > 0 {
		m.Reset()
		m.clientID, err = strconv.Atoi(match[1])
		m.keyID, err = strconv.Atoi(match[2])
		return false, true, nil
	}
	return true, false, nil
}

func (m *middleware) checkConnect(line string) (cont bool, consumed bool, err error) {

	rule, err := regexp.Compile("^>CLIENT:CONNECT,(\\d),(\\d)$")
	if err != nil {
		return false, false, err
	}

	match := rule.FindStringSubmatch(line)
	if len(match) == 0 {
		return true, false, nil
	}
	m.Reset()
	m.clientID, err = strconv.Atoi(match[1])
	m.keyID, err = strconv.Atoi(match[2])
	return false, true, nil

}

func (m *middleware) checkPassword(line string) (cont bool, consumed bool, err error) {

	rule, err := regexp.Compile("^>CLIENT:ENV,password=(.*)$")
	if err != nil {
		return false, false, err
	}

	match := rule.FindStringSubmatch(line)
	if len(match) > 0 {
		if m.clientID < 0 {
			return false, false, fmt.Errorf("wrong auth state, no client id")
		}
		m.lastPassword = match[1]
		return false, true, nil
	}

	return true, false, nil
}

func (m *middleware) checkUsername(line string) (cont bool, consumed bool, err error) {

	rule, err := regexp.Compile("^>CLIENT:ENV,username=(.*)$")
	if err != nil {
		return false, false, err
	}

	match := rule.FindStringSubmatch(line)
	if len(match) > 0 {
		if m.clientID < 0 {
			return false, false, fmt.Errorf("wrong auth state, no client id")
		}
		m.lastUsername = match[1]
		return false, true, nil
	}

	return true, false, nil
}

func (m *middleware) checkEnvEnd(line string) (cont bool, consumed bool, err error) {

	rule, err := regexp.Compile("^>CLIENT:ENV,END$")
	if err != nil {
		return false, false, err
	}

	match := rule.FindStringSubmatch(line)
	if len(match) > 0 {
		return false, true, nil
	}

	return true, false, nil
}

func (m *middleware) ConsumeLine(line string) (consumed bool, err error) {
	if cont, consumed, err := m.checkReAuth(line); !cont {
		return consumed, err
	}

	if cont, consumed, err := m.checkConnect(line); !cont {
		return consumed, err
	}

	if cont, consumed, err := m.checkUsername(line); !cont {
		return consumed, err
	}

	if cont, consumed, err := m.checkPassword(line); !cont {
		return consumed, err
	}

	if cont, consumed, err := m.checkEnvEnd(line); !cont {
		if consumed {
			return m.authenticateClient()
		}
		return consumed, err
	}

	return false, err
}

func (m *middleware) authenticateClient() (consumed bool, err error) {

	if m.lastUsername == "" || m.lastPassword == "" {
		denyClientAuthWithMessage(m.commandWriter, m.clientID, m.keyID, "missing username or password")
		return true, nil
	}

	log.Info("authenticating user: ", m.lastUsername, " clientID: ", m.clientID, " keyID: ", m.keyID)

	authenticated, err := m.checkCredentials(m.lastUsername, m.lastPassword)
	if err != nil {
		log.Error("Authentication error: ", err)
		denyClientAuthWithMessage(m.commandWriter, m.clientID, m.keyID, "internal error")
		return true, nil
	}

	if authenticated {
		approveClient(m.commandWriter, m.clientID, m.keyID)
	} else {
		denyClientAuthWithMessage(m.commandWriter, m.clientID, m.keyID, "wrong username or password")
	}
	return true, nil
}

func approveClient(commandWriter openvpn.CommandWriter, clientID, keyID int) error {
	return commandWriter.PrintfLine("client-auth-nt %d %d", clientID, keyID)
}

func denyClientAuthWithMessage(commandWriter openvpn.CommandWriter, clientID, keyID int, message string) error {
	return commandWriter.PrintfLine("client-deny %d %d %s", clientID, keyID, message)
}

func (m *middleware) Reset() {
	m.lastUsername = ""
	m.lastPassword = ""
	m.clientID = -1
	m.keyID = -1
}
