package communication

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBytesPack(t *testing.T) {
	packer := BytesPayload{[]byte("123")}
	data, err := packer.Pack()

	assert.NoError(t, err)
	assert.Equal(t, "123", string(data))
}

func TestBytesUnpack(t *testing.T) {
	var unpacker BytesPayload
	err := unpacker.Unpack([]byte("123"))

	assert.NoError(t, err)
	assert.Equal(t, "123", string(unpacker.Data))
}

func TestBytesListener(t *testing.T) {
	var messageConsumed *BytesPayload
	listener := BytesListener(func(message *BytesPayload) {
		messageConsumed = message
	})

	err := listener.Message.Unpack([]byte("123"))
	listener.Invoke()

	assert.NoError(t, err)
	assert.Equal(t, "123", string(messageConsumed.Data))
}

func TestBytesHandler(t *testing.T) {
	var requestReceived *BytesPayload
	handler := BytesHandler(func(request *BytesPayload) *BytesPayload {
		requestReceived = request
		return &BytesPayload{[]byte("RESPONSE")}
	})
	response := handler([]byte("REQUEST"))

	assert.Equal(t, "REQUEST", string(requestReceived.Data))
	assert.Equal(t, "RESPONSE", string(response))
}
