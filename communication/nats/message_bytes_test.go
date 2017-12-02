package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

type bytesMessagePacker struct {
	Message []byte
}

func (packer *bytesMessagePacker) GetMessageType() communication.MessageType {
	return communication.MessageType("bytes-message")
}

func (packer *bytesMessagePacker) CreateMessage() (messagePtr interface{}) {
	return packer.Message
}

type bytesMessageUnpacker struct {
	Callback func(*[]byte)
}

func (unpacker *bytesMessageUnpacker) GetMessageType() communication.MessageType {
	return communication.MessageType("bytes-message")
}

func (unpacker *bytesMessageUnpacker) CreateMessage() (messagePtr interface{}) {
	var message []byte
	return &message
}

func (unpacker *bytesMessageUnpacker) Handle(messagePtr interface{}) error {
	unpacker.Callback(messagePtr.(*[]byte))
	return nil
}

func TestMessageBytesSend(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{
		connection: connection,
		codec:      communication.NewCodecBytes(),
	}

	messageSent := make(chan bool)
	_, err := connection.Subscribe("bytes-message", func(message *nats.Msg) {
		assert.Equal(t, []byte("123"), message.Data)
		messageSent <- true
	})
	assert.Nil(t, err)

	err = sender.Send(
		&bytesMessagePacker{[]byte("123")},
	)
	assert.Nil(t, err)

	if err := test.Wait(messageSent); err != nil {
		t.Fatal("Message not sent")
	}
}

func TestMessageBytesReceive(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{
		connection: connection,
		codec:      communication.NewCodecBytes(),
	}

	messageReceived := make(chan bool)
	err := receiver.Receive(&bytesMessageUnpacker{func(message *[]byte) {
		assert.Equal(t, []byte("123"), *message)
		messageReceived <- true
	}})
	assert.Nil(t, err)

	err = connection.Publish("bytes-message", []byte("123"))
	assert.Nil(t, err)

	if err := test.Wait(messageReceived); err != nil {
		t.Fatal("Message not received")
	}
}
