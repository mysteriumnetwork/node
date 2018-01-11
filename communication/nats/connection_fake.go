package nats

import (
	"fmt"
	"github.com/nats-io/go-nats"
	"github.com/pkg/errors"
	"time"
)

// NewConnectionFake constructs new NATS connection
// which delivers published messages to local subscribers
func NewConnectionFake() *connectionFake {
	return &connectionFake{
		subscriptions: make(map[string][]nats.MsgHandler),
		queue:         make(chan *nats.Msg),
		queueShutdown: make(chan bool),
	}
}

// StartConnectionFake creates connection and starts it immediately
func StartConnectionFake() *connectionFake {
	connection := NewConnectionFake()
	connection.Start()

	return connection
}

type connectionFake struct {
	subscriptions map[string][]nats.MsgHandler
	queue         chan *nats.Msg
	queueShutdown chan bool

	messageLast *nats.Msg
	requestLast *nats.Msg
	errorMock   error
}

func (conn *connectionFake) GetLastMessage() []byte {
	if conn.messageLast != nil {
		return conn.messageLast.Data
	}
	return []byte{}
}

func (conn *connectionFake) GetLastRequest() []byte {
	if conn.requestLast != nil {
		return conn.requestLast.Data
	}
	return []byte{}
}

func (conn *connectionFake) MockResponse(subject string, payload []byte) {
	conn.Subscribe(subject, func(message *nats.Msg) {
		conn.Publish(message.Reply, payload)
	})
}

func (conn *connectionFake) MockError(message string) {
	conn.errorMock = errors.New(message)
}

func (conn *connectionFake) MessageWait(waitChannel chan interface{}) (interface{}, error) {
	select {
	case message := <-waitChannel:
		return message, nil
	case <-time.After(10 * time.Millisecond):
		return nil, errors.New("Message not received")
	}
}

func (conn *connectionFake) Publish(subject string, payload []byte) error {
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

func (conn *connectionFake) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if conn.errorMock != nil {
		return nil, conn.errorMock
	}

	conn.subscriptionAdd(subject, handler)

	return &nats.Subscription{}, nil
}

func (conn *connectionFake) Request(subject string, payload []byte, timeout time.Duration) (*nats.Msg, error) {
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
		return nil, fmt.Errorf("Request '%s' timeout", subject)
	}
}

func (conn *connectionFake) Start() {
	go conn.queueProcessing()
}

func (conn *connectionFake) Close() {
	conn.queueShutdown <- true
}

func (conn *connectionFake) subscriptionAdd(subject string, handler nats.MsgHandler) {
	subscriptions, exist := conn.subscriptions[subject]
	if exist {
		subscriptions = append(subscriptions, handler)
	} else {
		conn.subscriptions[subject] = []nats.MsgHandler{handler}
	}
}

func (conn *connectionFake) subscriptionsGet(subject string) (*[]nats.MsgHandler, bool) {
	subscriptions, exist := conn.subscriptions[subject]
	return &subscriptions, exist
}

func (conn *connectionFake) queueProcessing() {
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
