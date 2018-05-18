/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package auth

import (
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/openvpn/management"
	"strings"
)

// CredentialsChecker callback checks given auth primitives (i.e. customer identity signature / node's sessionId)
type CredentialsChecker func(username, password string) (bool, error)

type middleware struct {
	checkCredentials CredentialsChecker
	commandWriter    management.Connection
	currentEvent     clientEvent
}

// NewMiddleware creates server user_auth challenge authentication middleware
func NewMiddleware(credentialsChecker CredentialsChecker) *middleware {
	return &middleware{
		checkCredentials: credentialsChecker,
		commandWriter:    nil,
		currentEvent:     undefinedEvent,
	}
}

type clientEventType string

const (
	connect     = clientEventType("CONNECT")
	reauth      = clientEventType("REAUTH")
	established = clientEventType("ESTABLISHED")
	disconnect  = clientEventType("DISCONNECT")
	address     = clientEventType("ADDRESS")
	//pseudo event type ENV - that means some of above defined events are multiline and ENV messages are part of it
	env = clientEventType("ENV")
	//constant which means that id of type int is undefined
	undefined = -1
)

type clientEvent struct {
	eventType clientEventType
	clientID  int
	clientKey int
	env       map[string]string
}

var undefinedEvent = clientEvent{
	clientID:  undefined,
	clientKey: undefined,
	env:       make(map[string]string),
}

func (m *middleware) Start(commandWriter management.Connection) error {
	m.commandWriter = commandWriter
	return nil
}

func (m *middleware) Stop(commandWriter management.Connection) error {
	return nil
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
		ID, key, err := parseIDAndKey(eventData)
		if err != nil {
			return true, err
		}
		m.startOfEvent(eventType, ID, key)
	case env:
		if strings.ToLower(eventData) == "end" {
			m.endOfEvent()
			return true, nil
		}

		key, val, err := parseEnvVar(eventData)
		if err != nil {
			return true, err
		}
		m.addEnvVar(key, val)
	case established, disconnect:
		ID, err := parseID(eventData)
		if err != nil {
			return true, err
		}
		m.startOfEvent(eventType, ID, undefined)
	case address:
		log.Info("Address for client: ", eventData)
	default:
		log.Error("Undefined user notification event: ", eventType, eventData)
		log.Error("Original line was: ", line)
	}
	return true, nil
}

func (m *middleware) startOfEvent(eventType clientEventType, clientID int, keyID int) {
	m.currentEvent.eventType = eventType
	m.currentEvent.clientID = clientID
	m.currentEvent.clientKey = keyID
}

func (m *middleware) addEnvVar(key string, val string) {
	m.currentEvent.env[key] = val
}

func (m *middleware) endOfEvent() {
	m.handleClientEvent(m.currentEvent)
	m.reset()
}

func (m *middleware) reset() {
	m.currentEvent = undefinedEvent
}

func (m *middleware) handleClientEvent(event clientEvent) {
	switch event.eventType {
	case connect, reauth:
		username := event.env["username"]
		password := event.env["password"]
		err := m.authenticateClient(event.clientID, event.clientKey, username, password)
		if err != nil {
			log.Error("Unable to authenticate client. Error: ", err)
		}
	case established:
		log.Info("Client with ID: ", event.clientID, " connection established successfully")
	case disconnect:
		log.Info("Client with ID: ", event.clientID, " disconnected")
	}
}

func (m *middleware) authenticateClient(clientID, clientKey int, username, password string) error {

	if username == "" || password == "" {
		return denyClientAuthWithMessage(m.commandWriter, clientID, clientKey, "missing username or password")
	}

	log.Info("authenticating user: ", username, " clientID: ", clientID, " clientKey: ", clientKey)

	authenticated, err := m.checkCredentials(username, password)
	if err != nil {
		log.Error("Authentication error: ", err)
		return denyClientAuthWithMessage(m.commandWriter, clientID, clientKey, "internal error")
	}

	if authenticated {
		return approveClient(m.commandWriter, clientID, clientKey)
	}
	return denyClientAuthWithMessage(m.commandWriter, clientID, clientKey, "wrong username or password")
}

func approveClient(commandWriter management.Connection, clientID, keyID int) error {
	_, err := commandWriter.SingleLineCommand("client-auth-nt %d %d", clientID, keyID)
	return err
}

func denyClientAuthWithMessage(commandWriter management.Connection, clientID, keyID int, message string) error {
	_, err := commandWriter.SingleLineCommand("client-deny %d %d %s", clientID, keyID, message)
	return err
}
