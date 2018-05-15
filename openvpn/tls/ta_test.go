package tls

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const expectedKey = `-----BEGIN OpenVPN Static key V1-----
616263
-----END OpenVPN Static key V1-----
`

func TestTLSPresharedKeyProducesValidPEMFormat(t *testing.T) {
	key := TLSPresharedKey("abc")
	assert.Equal(
		t,
		expectedKey,
		key.ToPEMFormat(),
	)
}

func TestGeneratedKeyIsExpectedSize(t *testing.T) {
	key, err := createTLSCryptKey()
	assert.NoError(t, err)
	assert.Len(t, key, 256)
}
