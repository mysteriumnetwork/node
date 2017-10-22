package communication

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStringPack(t *testing.T) {
	packer := StringPayload{"123"}
	data, err := packer.Pack()

	assert.NoError(t, err)
	assert.Equal(t, "123", string(data))
}

func TestStringUnpack(t *testing.T) {
	var unpacker StringPayload
	err := unpacker.Unpack([]byte("123"))

	assert.NoError(t, err)
	assert.Equal(t, "123", string(unpacker.Data))
}

func TestStringListener(t *testing.T) {
	var messageConsumed *StringPayload
	listener := StringListener(func(message *StringPayload) {
		messageConsumed = message
	})

	err := listener.Message.Unpack([]byte("123"))
	listener.Invoke()

	assert.NoError(t, err)
	assert.Equal(t, &StringPayload{"123"}, messageConsumed)
}

func TestStringHandler(t *testing.T) {
	var requestReceived *StringPayload
	handler := StringHandler(func(request *StringPayload) *StringPayload {
		requestReceived = request
		return &StringPayload{"RESPONSE"}
	})

	err := handler.Request.Unpack([]byte("REQUEST"))
	response := handler.Invoke()

	assert.NoError(t, err)
	assert.Equal(t, &StringPayload{"REQUEST"}, requestReceived)
	assert.Equal(t, &StringPayload{"RESPONSE"}, response)
}
