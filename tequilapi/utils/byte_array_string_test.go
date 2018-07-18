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

	assert.Equal(t, "{\"TestString\":\"0x00010203\"}", string(stringBytes))
}
