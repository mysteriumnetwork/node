package communication

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStringProduce(t *testing.T) {
	producer := StringProduce{"123"}

	assert.Equal(t, "123", string(producer.ProduceMessage()))
}

func TestStringResponse(t *testing.T) {
	consumer := StringResponse{}
	consumer.ConsumeMessage([]byte("123"))

	assert.Equal(t, "123", consumer.Response)
}

func TestStringCallback(t *testing.T) {
	var messageConsumed string
	consumer := StringCallback{func(message string) {
		messageConsumed = message
	}}
	consumer.ConsumeMessage([]byte("123"))

	assert.Equal(t, "123", messageConsumed)
}
