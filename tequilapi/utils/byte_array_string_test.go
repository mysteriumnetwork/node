package utils

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestByteArrayStringMarshalsToHexString(t *testing.T) {
	bytes := []byte{0, 1, 2, 3}

	stringBytes, err := json.Marshal(
		struct {
			TestString *ByteArrayString
		}{
			ToByteArrayString(bytes),
		},
	)
	assert.NoError(t, err)

	assert.Equal(t, "{\"TestString\":\"\\\\x00\\\\x01\\\\x02\\\\x03\"}", string(stringBytes))
}
