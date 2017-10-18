package communication

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStringPack(t *testing.T) {
	packer := StringPayload{"123"}
	data := packer.Pack()

	assert.Equal(t, "123", string(data))
}

func TestStringUnpack(t *testing.T) {
	var unpacker StringPayload
	unpacker.Unpack([]byte("123"))

	assert.Equal(t, "123", string(unpacker.Data))
}

func TestStringListener(t *testing.T) {
	var messageConsumed *StringPayload
	listener := StringListener(func(message *StringPayload) {
		messageConsumed = message
	})
	listener([]byte("123"))

	assert.Equal(t, "123", messageConsumed.Data)
}

func TestStringHandler(t *testing.T) {
	var requestReceived *StringPayload
	handler := StringHandler(func(request *StringPayload) *StringPayload {
		requestReceived = request
		return &StringPayload{"RESPONSE"}
	})
	response := handler([]byte("REQUEST"))

	assert.Equal(t, "REQUEST", requestReceived.Data)
	assert.Equal(t, "RESPONSE", string(response))
}
