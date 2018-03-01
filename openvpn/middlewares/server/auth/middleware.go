package auth

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/openvpn/management"
	"regexp"
	"strconv"
	"strings"
)

// CredentialsChecker callback checks given auth primitives (i.e. customer identity signature / node's sessionId)
type CredentialsChecker func(username, password string) (bool, error)

type middleware struct {
	checkCredentials CredentialsChecker
	commandWriter    management.Connection
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
		clientID:         undefined,
		keyID:            undefined,
	}
}

type clientEvent string

const (
	connect     = clientEvent("CONNECT")
	reauth      = clientEvent("REAUTH")
	established = clientEvent("ESTABLISHED")
	disconnect  = clientEvent("DISCONNECT")
	address     = clientEvent("ADDRESS")
	//pseudo event type ENV - that means some of above defined events are multiline and ENV messages are part of it
	env = clientEvent("ENV")
	//constant which means that id of type int is undefined
	undefined = -1
)

func (m *middleware) Start(commandWriter management.Connection) error {
	m.commandWriter = commandWriter
	return nil
}

func (m *middleware) Stop(commandWriter management.Connection) error {
	return nil
}

func (m *middleware) checkReAuth(line string) (cont bool, consumed bool, err error) {

	rule, err := regexp.Compile("^>CLIENT:REAUTH,(\\d+),(\\d+)$")
	if err != nil {
		return false, false, err
	}

	match := rule.FindStringSubmatch(line)
	if len(match) > 0 {
		m.reset()
		m.clientID, err = strconv.Atoi(match[1])
		m.keyID, err = strconv.Atoi(match[2])
		return false, true, nil
	}
	return true, false, nil
}

func (m *middleware) checkConnect(line string) (cont bool, consumed bool, err error) {

	rule, err := regexp.Compile("^>CLIENT:CONNECT,(\\d+),(\\d+)$")
	if err != nil {
		return false, false, err
	}

	match := rule.FindStringSubmatch(line)
	if len(match) == 0 {
		return true, false, nil
	}
	m.reset()
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

func (m *middleware) ConsumeLine(line string) (bool, error) {
	if !strings.HasPrefix(line, ">CLIENT:") {
		return false, nil
	}

	clientLine := strings.TrimPrefix(line, ">CLIENT:")

	eventType, eventData, err := parseClientEvent(clientLine)
	if err != nil {
		return true, err
	}

	switch eventType {
	case connect, reauth:

	case env:
		if strings.ToLower(eventData) == "end" {
			return true, nil
		}

		key, val, err := parseEnvVar(eventData)
		if err != nil {
			return true, err
		}
		m.addEnvVar(key, val)
	case established, disconnect:

	case address:
		log.Info("Address for client: ", eventData)
	default:
		log.Error("Undefined user notification event: ", eventType, eventData)
		log.Error("Original line was: ", line)
	}
	return true, nil
}

func (m *middleware) authenticateClient() error {

	if m.lastUsername == "" || m.lastPassword == "" {
		return denyClientAuthWithMessage(m.commandWriter, m.clientID, m.keyID, "missing username or password")
	}

	log.Info("authenticating user: ", m.lastUsername, " clientID: ", m.clientID, " keyID: ", m.keyID)

	authenticated, err := m.checkCredentials(m.lastUsername, m.lastPassword)
	if err != nil {
		log.Error("Authentication error: ", err)
		return denyClientAuthWithMessage(m.commandWriter, m.clientID, m.keyID, "internal error")
	}

	if authenticated {
		return approveClient(m.commandWriter, m.clientID, m.keyID)
	}
	return denyClientAuthWithMessage(m.commandWriter, m.clientID, m.keyID, "wrong username or password")
}

func approveClient(commandWriter management.Connection, clientID, keyID int) error {
	_, err := commandWriter.SingleLineCommand("client-auth-nt %d %d", clientID, keyID)
	return err
}

func denyClientAuthWithMessage(commandWriter management.Connection, clientID, keyID int, message string) error {
	_, err := commandWriter.SingleLineCommand("client-deny %d %d %s", clientID, keyID, message)
	return err
}

func (m *middleware) reset() {
	m.clientID = undefined
	m.keyID = undefined
}

func (middleware *middleware) addEnvVar(key string, val string) {

}
