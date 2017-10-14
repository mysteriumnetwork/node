package communication

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBytesPack(t *testing.T) {
	packer := BytesPacker([]byte("123"))
	data := packer()

	assert.Equal(t, "123", string(data))
}

func TestBytesUnpack(t *testing.T) {
	var message []byte
	unpacker := BytesUnpacker(&message)
	unpacker([]byte("123"))

	assert.Equal(t, "123", string(message))
}

func TestBytesListener(t *testing.T) {
	var messageConsumed []byte
	listener := BytesListener(func(message []byte) {
		messageConsumed = message
	})
	listener([]byte("123"))

	assert.Equal(t, "123", string(messageConsumed))
}
