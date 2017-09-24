package nats

import (
	"fmt"
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"
)

type serverNats struct {
	myTopic        string
	options        nats.Options
	timeoutRequest time.Duration

	connection    *nats.Conn
	dialogHandler communication.DialogHandler
}

func (server *serverNats) ServeDialogs(dialogHandler communication.DialogHandler) error {
	if err := server.Start(); err != nil {
		return err
	}
	server.dialogHandler = dialogHandler

	_, err := server.connection.Subscribe(
		server.myTopic+"."+string(communication.DIALOG_CREATE),
		server.acceptDialog,
	)
	return err
}

func (server *serverNats) acceptDialog(message *nats.Msg) {
	request := string(message.Data)

	response := "OK"
	if err := server.connection.Publish(message.Reply, []byte(response)); err != nil {
		return
	}

	server.createDialog(request)
}

func (server *serverNats) createDialog(receiverTopic string) {
	fmt.Printf("Dialog requested. topic=%s\n", receiverTopic)

	sender := &senderNats{
		connection:     server.connection,
		receiverTopic:  server.myTopic,
		timeoutRequest: server.timeoutRequest,
	}
	receiver := &receiverNats{
		connection:    server.connection,
		receiverTopic: server.myTopic,
	}
	server.dialogHandler(sender, receiver)
}

func (server *serverNats) Start() (err error) {
	server.connection, err = server.options.Connect()
	return err
}

func (server *serverNats) Stop() error {
	server.connection.Close()
	return nil
}
