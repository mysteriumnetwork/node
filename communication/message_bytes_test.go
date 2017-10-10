package communication

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBytesProduce(t *testing.T) {
	producer := BytesProduce{[]byte("123")}

	assert.Equal(t, "123", string(producer.ProduceMessage()))
}

func TestBytesCallback(t *testing.T) {
	var messageConsumed []byte
	producer := BytesCallback{func(message []byte) {
		messageConsumed = message
	}}
	producer.ConsumeMessage([]byte("123"))

	assert.Equal(t, "123", string(messageConsumed))
}
