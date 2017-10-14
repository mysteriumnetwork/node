package communication

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStringPack(t *testing.T) {
	packer := StringPacker("123")
	data := packer()

	assert.Equal(t, "123", string(data))
}

func TestStringUnpack(t *testing.T) {
	var message string
	unpacker := StringUnpacker(&message)
	unpacker([]byte("123"))

	assert.Equal(t, "123", message)
}

func TestStringListener(t *testing.T) {
	var messageConsumed string
	listener := StringListener(func(message string) {
		messageConsumed = message
	})
	listener([]byte("123"))

	assert.Equal(t, "123", messageConsumed)
}

func TestStringHandler(t *testing.T) {
	var requestReceived string
	handler := StringHandler(func(request string) string {
		requestReceived = request
		return "RESPONSE"
	})
	response := handler([]byte("REQUEST"))

	assert.Equal(t, "REQUEST", requestReceived)
	assert.Equal(t, "RESPONSE", string(response))
}
