/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package nats

import (
	"time"

	"github.com/nats-io/go-nats"
	"github.com/pkg/errors"
)

// NewConnectionMock constructs new NATS connection
// which delivers published messages to local subscribers
func NewConnectionMock() *ConnectionMock {
	return &ConnectionMock{
		subscriptions: make(map[string][]nats.MsgHandler),
		queue:         make(chan *nats.Msg),
		queueShutdown: make(chan bool),
	}
}

// StartConnectionMock creates connection and starts it immediately
func StartConnectionMock() *ConnectionMock {
	connection := NewConnectionMock()
	connection.Open()

	return connection
}

// ConnectionMock acts as a local connection implementation
type ConnectionMock struct {
	subscriptions map[string][]nats.MsgHandler
	queue         chan *nats.Msg
	queueShutdown chan bool

	messageLast *nats.Msg
	requestLast *nats.Msg
	errorMock   error
}

// GetLastMessageSubject returns the last message subject
func (conn *ConnectionMock) GetLastMessageSubject() string {
	if conn.messageLast != nil {
		return conn.messageLast.Subject
	}
	return ""
}

// GetLastMessage returns the last message received
func (conn *ConnectionMock) GetLastMessage() []byte {
	if conn.messageLast != nil {
		return conn.messageLast.Data
	}
	return []byte{}
}

// GetLastRequest gets last request data
func (conn *ConnectionMock) GetLastRequest() []byte {
	if conn.requestLast != nil {
		return conn.requestLast.Data
	}
	return []byte{}
}

// MockResponse mocks the response
func (conn *ConnectionMock) MockResponse(subject string, payload []byte) {
	conn.Subscribe(subject, func(message *nats.Msg) {
		conn.Publish(message.Reply, payload)
	})
}

// MockError mocks the error
func (conn *ConnectionMock) MockError(message string) {
	conn.errorMock = errors.New(message)
}

// MessageWait waits for a message to arrive
func (conn *ConnectionMock) MessageWait(waitChannel chan interface{}) (interface{}, error) {
	select {
	case message := <-waitChannel:
		return message, nil
	case <-time.After(10 * time.Millisecond):
		return nil, errors.New("Message not received")
	}
}

// Publish publishes a new message
func (conn *ConnectionMock) Publish(subject string, payload []byte) error {
	if conn.errorMock != nil {
		return conn.errorMock
	}

	conn.messageLast = &nats.Msg{
		Subject: subject,
		Data:    payload,
	}
	conn.queue <- conn.messageLast

	return nil
}

// Subscribe subscribes to a topic
func (conn *ConnectionMock) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if conn.errorMock != nil {
		return nil, conn.errorMock
	}

	conn.subscriptionAdd(subject, handler)

	return &nats.Subscription{}, nil
}

// Request sends a new request
func (conn *ConnectionMock) Request(subject string, payload []byte, timeout time.Duration) (*nats.Msg, error) {
	if conn.errorMock != nil {
		return nil, conn.errorMock
	}

	subjectReply := subject + "-reply"
	responseCh := make(chan *nats.Msg)
	conn.Subscribe(subjectReply, func(response *nats.Msg) {
		responseCh <- response
	})

	conn.requestLast = &nats.Msg{
		Subject: subject,
		Reply:   subjectReply,
		Data:    payload,
	}
	conn.queue <- conn.requestLast

	select {
	case response := <-responseCh:
		return response, nil
	case <-time.After(timeout):
		return nil, errors.Errorf("request '%s' timeout", subject)
	}
}

// Open starts the connection
func (conn *ConnectionMock) Open() error {
	go conn.queueProcessing()
	return nil
}

// Close destructs the connection
func (conn *ConnectionMock) Close() {
	conn.queueShutdown <- true
}

// Check checks the connection
func (conn *ConnectionMock) Check() error {
	return nil
}

// Servers returns list of currently connected servers
func (conn *ConnectionMock) Servers() []string {
	return []string{"mockhost"}
}

func (conn *ConnectionMock) subscriptionAdd(subject string, handler nats.MsgHandler) {
	subscriptions, exist := conn.subscriptions[subject]
	if exist {
		subscriptions = append(subscriptions, handler)
	} else {
		conn.subscriptions[subject] = []nats.MsgHandler{handler}
	}
}

func (conn *ConnectionMock) subscriptionsGet(subject string) (*[]nats.MsgHandler, bool) {
	subscriptions, exist := conn.subscriptions[subject]
	return &subscriptions, exist
}

func (conn *ConnectionMock) queueProcessing() {
	for {
		select {
		case <-conn.queueShutdown:
			break

		case message := <-conn.queue:
			if subscriptions, exist := conn.subscriptionsGet(message.Subject); exist {
				for _, handler := range *subscriptions {
					go handler(message)
				}
			}
		}
	}
}
