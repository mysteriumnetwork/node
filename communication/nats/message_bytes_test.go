package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func bytesMessagePacker(message []byte) communication.MessagePacker {
	return communication.MessagePacker{
		MessageType: "bytes-message",
		Pack: func() ([]byte, error) {
			return message, nil
		},
	}
}

func bytesMessageUnpacker(listener func([]byte)) communication.MessageUnpacker {
	var message []byte

	return communication.MessageUnpacker{
		MessageType: "bytes-message",
		Unpack: func(data []byte) error {
			message = data
			return nil
		},
		Invoke: func() error {
			listener(message)
			return nil
		},
	}
}

func TestMessageBytesSend(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{connection: connection}

	messageSent := make(chan bool)
	_, err := connection.Subscribe("bytes-message", func(message *nats.Msg) {
		assert.Equal(t, "123", string(message.Data))
		messageSent <- true
	})
	assert.Nil(t, err)

	err = sender.Send2(bytesMessagePacker([]byte("123")))
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

	receiver := &receiverNats{connection: connection}

	messageReceived := make(chan bool)
	err := receiver.Receive2(
		bytesMessageUnpacker(func(message []byte) {
			assert.Equal(t, "123", string(message))
			messageReceived <- true
		}),
	)
	assert.Nil(t, err)

	err = connection.Publish("bytes-message", []byte("123"))
	assert.Nil(t, err)

	if err := test.Wait(messageReceived); err != nil {
		t.Fatal("Message not received")
	}
}
