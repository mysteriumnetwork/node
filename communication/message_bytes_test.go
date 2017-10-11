package communication

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBytesProduce(t *testing.T) {
	producer := BytesProduce{[]byte("123")}

	assert.Equal(t, "123", string(producer.ProduceMessage()))
}

func TestBytesResponse(t *testing.T) {
	consumer := BytesResponse{}
	consumer.ConsumeMessage([]byte("123"))

	assert.Equal(t, "123", string(consumer.Response))
}

func TestBytesCallback(t *testing.T) {
	var messageConsumed []byte
	consumer := BytesCallback{func(message []byte) {
		messageConsumed = message
	}}
	consumer.ConsumeMessage([]byte("123"))

	assert.Equal(t, "123", string(messageConsumed))
}
