package auth

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/state"
	"net"
	"regexp"
	"strconv"
)

type middleware struct {
	authenticator Authenticator
	connection    net.Conn
	username      string
	password      string
	clientId      int
	keyId         int
	state         state.State
}

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

func (m *middleware) ConsumeLine(line string) (consumed bool, err error) {

	rule, err := regexp.Compile("^>CLIENT:REAUTH,(\\d),(\\d)$")
	if err != nil {
		return false, err
	}

	match := rule.FindStringSubmatch(line)
	if len(match) > 0 {
		m.Reset()
		m.state = state.STATE_AUTH
		m.clientId, err = strconv.Atoi(match[1])
		m.keyId, err = strconv.Atoi(match[2])
		return true, nil
	}

	rule, err = regexp.Compile("^>CLIENT:CONNECT,(\\d),(\\d)$")
	if err != nil {
		return false, err
	}

	match = rule.FindStringSubmatch(line)
	if len(match) > 0 {
		m.Reset()
		m.state = state.STATE_AUTH
		m.clientId, err = strconv.Atoi(match[1])
		m.keyId, err = strconv.Atoi(match[2])
		return true, nil
	}

	// further proceed only if in AUTH state
	if m.state != state.STATE_AUTH {
		return false, nil
	}

	rule, err = regexp.Compile("^>CLIENT:ENV,password=(.*)$")
	if err != nil {
		return false, err
	}

	match = rule.FindStringSubmatch(line)
	if len(match) > 0 {
		if m.clientId < 0 {
			return false, fmt.Errorf("wrong auth state")
		}
		m.password = match[1]
		return true, nil
	}

	rule, err = regexp.Compile("^>CLIENT:ENV,username=(.*)$")
	if err != nil {
		return false, err
	}

	match = rule.FindStringSubmatch(line)
	if len(match) > 0 {
		if m.clientId < 0 {
			return false, fmt.Errorf("wrong auth state")
		}
		m.username = match[1]
		return true, nil
	}

	rule, err = regexp.Compile("^>CLIENT:ENV,END$")
	if err != nil {
		return false, err
	}

	match = rule.FindStringSubmatch(line)
	if len(match) > 0 {
		defer m.Reset()

		if m.username == "" || m.password == "" {
			return false, fmt.Errorf("missing username or password")
		}

		log.Info("authenticating user: ", m.username, " clientId: ", m.clientId, " keyId: ", m.keyId)

		authenticated, err := m.authenticator(m.username, m.password)
		if err != nil {
			return false, err
		}

		if authenticated {
			_, err = m.connection.Write([]byte("client-auth-nt " + strconv.Itoa(m.clientId) + " " + strconv.Itoa(m.keyId) + "\n"))
			if err != nil {
				return false, err
			}
		} else {
			_, err = m.connection.Write([]byte("client-deny " + strconv.Itoa(m.clientId) + " " + strconv.Itoa(m.keyId) +
				" wrong username or password \n"))
			if err != nil {
				return false, err
			}
		}
		return true, nil
	}

	return false, nil
}
func (m *middleware) Reset() {
	m.username = ""
	m.password = ""
	m.clientId = -1
	m.keyId = -1
	m.state = state.STATE_WAIT
}
