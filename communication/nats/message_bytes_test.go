package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/stretchr/testify/assert"
	"testing"
)

type bytesMessageProducer struct {
	Message []byte
}

func (producer *bytesMessageProducer) GetMessageType() communication.MessageType {
	return communication.MessageType("bytes-message")
}

func (producer *bytesMessageProducer) Produce() (message interface{}) {
	return producer.Message
}

type bytesMessageConsumer struct {
	messageReceived chan interface{}
}

func (consumer *bytesMessageConsumer) GetMessageType() communication.MessageType {
	return communication.MessageType("bytes-message")
}

func (consumer *bytesMessageConsumer) NewMessage() interface{} {
	var message []byte
	return &message
}

func (consumer *bytesMessageConsumer) Consume(message interface{}) error {
	consumer.messageReceived <- message
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
	// assert.Equal(t, []byte("123"), message.([]byte))
	// assert.Equal(t, []byte("123"), *message.(*[]byte))
	expected := []byte("123")
	assert.Equal(t, &expected, message)
}
