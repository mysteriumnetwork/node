package communication

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStringProduce(t *testing.T) {
	producer := StringProduce{"123"}

	assert.Equal(t, "123", string(producer.ProduceMessage()))
}

func TestStringCallback(t *testing.T) {
	var messageConsumed string
	producer := StringCallback{func(message string) {
		messageConsumed = message
	}}
	producer.ConsumeMessage([]byte("123"))

	assert.Equal(t, "123", messageConsumed)
}
