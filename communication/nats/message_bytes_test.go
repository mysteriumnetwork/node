package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/stretchr/testify/assert"
	"testing"
)

type bytesMessageProducer struct {
	Message []byte
}

func (producer *bytesMessageProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return communication.MessageEndpoint("bytes-message")
}

func (producer *bytesMessageProducer) Produce() (messagePtr interface{}) {
	return producer.Message
}

type bytesMessageConsumer struct {
	messageReceived chan interface{}
}

func (consumer *bytesMessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return communication.MessageEndpoint("bytes-message")
}

func (consumer *bytesMessageConsumer) NewMessage() (messagePtr interface{}) {
	var message []byte
	return &message
}

func (consumer *bytesMessageConsumer) Consume(messagePtr interface{}) error {
	consumer.messageReceived <- messagePtr
	return nil
}

func TestMessageBytesSend(t *testing.T) {
	connection := StartConnectionFake()
	defer connection.Close()

	sender := &senderNats{
		connection: connection,
		codec:      communication.NewCodecBytes(),
	}

	err := sender.Send(
		&bytesMessageProducer{[]byte("123")},
	)
	assert.NoError(t, err)
	assert.Equal(t, []byte("123"), connection.GetLastMessage())
}

func TestMessageBytesReceive(t *testing.T) {
	connection := StartConnectionFake()
	defer connection.Close()

	receiver := &receiverNats{
		connection: connection,
		codec:      communication.NewCodecBytes(),
	}

	consumer := &bytesMessageConsumer{messageReceived: make(chan interface{})}
	err := receiver.Receive(consumer)
	assert.NoError(t, err)

	connection.Publish("bytes-message", []byte("123"))
	message, err := connection.MessageWait(consumer.messageReceived)
	assert.NoError(t, err)
	assert.Equal(t, []byte("123"), *message.(*[]byte))
}
