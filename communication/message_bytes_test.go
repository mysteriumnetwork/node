package communication

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBytesProduce(t *testing.T) {
	packer := BytesPacker([]byte("123"))
	assert.Equal(t, "123", string(packer()))
}

func TestBytesResponse(t *testing.T) {
	var response []byte
	unpacker := BytesUnpacker(&response)
	unpacker([]byte("123"))

	assert.Equal(t, "123", string(response))
}
