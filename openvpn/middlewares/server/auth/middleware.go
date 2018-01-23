package auth

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/openvpn"
	"net"
	"regexp"
	"strconv"
)

type middleware struct {
	authenticator Authenticator
	connection    net.Conn
	lastUsername  string
	lastPassword  string
	clientID      int
	keyID         int
	state         openvpn.State
}

// NewMiddleware creates server user_auth challenge authentication middleware
func NewMiddleware(authenticator Authenticator) openvpn.ManagementMiddleware {
	return &middleware{
		authenticator: authenticator,
		connection:    nil,
	}
}

func (m *middleware) Start(connection net.Conn) error {
	m.connection = connection

	_, err := m.connection.Write([]byte("state on all\n"))
	return err
}

func (m *middleware) Stop() error {
	_, err := m.connection.Write([]byte("state off all\n"))
	return err
}

func (m *middleware) State() openvpn.State {
	return m.state
}

func (m *middleware) ConsumeLine(line string) (consumed bool, err error) {

	rule, err := regexp.Compile("^>CLIENT:REAUTH,(\\d),(\\d)$")
	if err != nil {
		return false, err
	}

	match := rule.FindStringSubmatch(line)
	if len(match) > 0 {
		m.Reset()
		m.state = openvpn.STATE_AUTH
		m.clientID, err = strconv.Atoi(match[1])
		m.keyID, err = strconv.Atoi(match[2])
		return true, nil
	}

	rule, err = regexp.Compile("^>CLIENT:CONNECT,(\\d),(\\d)$")
	if err != nil {
		return false, err
	}

	match = rule.FindStringSubmatch(line)
	if len(match) > 0 {
		m.Reset()
		m.state = openvpn.STATE_AUTH
		m.clientID, err = strconv.Atoi(match[1])
		m.keyID, err = strconv.Atoi(match[2])
		return true, nil
	}

	// further proceed only if in AUTH state
	if m.state != openvpn.STATE_AUTH {
		return false, nil
	}

	rule, err = regexp.Compile("^>CLIENT:ENV,password=(.*)$")
	if err != nil {
		return false, err
	}

	match = rule.FindStringSubmatch(line)
	if len(match) > 0 {
		if m.clientID < 0 {
			return false, fmt.Errorf("wrong auth state, no client id")
		}
		m.lastPassword = match[1]
		return true, nil
	}

	rule, err = regexp.Compile("^>CLIENT:ENV,username=(.*)$")
	if err != nil {
		return false, err
	}

	match = rule.FindStringSubmatch(line)
	if len(match) > 0 {
		if m.clientID < 0 {
			return false, fmt.Errorf("wrong auth state, no client id")
		}
		m.lastUsername = match[1]
		return true, nil
	}

	rule, err = regexp.Compile("^>CLIENT:ENV,END$")
	if err != nil {
		return false, err
	}

	match = rule.FindStringSubmatch(line)
	if len(match) > 0 {
		return m.authenticateClient()
	}

	return false, nil
}

func (m *middleware) authenticateClient() (consumed bool, err error) {
	defer m.Reset()

	if m.lastUsername == "" || m.lastPassword == "" {
		return false, fmt.Errorf("missing username or password")
	}

	log.Info("authenticating user: ", m.lastUsername, " clientID: ", m.clientID, " keyID: ", m.keyID)

	authenticated, err := m.authenticator(m.lastUsername, m.lastPassword)
	if err != nil {
		return false, err
	}

	if authenticated {
		_, err = m.connection.Write([]byte("client-auth-nt " + strconv.Itoa(m.clientID) + " " + strconv.Itoa(m.keyID) + "\n"))
		if err != nil {
			return false, err
		}
	} else {
		_, err = m.connection.Write([]byte("client-deny " + strconv.Itoa(m.clientID) + " " + strconv.Itoa(m.keyID) +
			" wrong username or password \n"))
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (m *middleware) Reset() {
	m.lastUsername = ""
	m.lastPassword = ""
	m.clientID = -1
	m.keyID = -1
	m.state = openvpn.STATE_UNDEFINED
}
