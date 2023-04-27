package pkce

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPKCEInfo(t *testing.T) {
	_, err := New(42)
	assert.Error(t, err)

	_, err = New(129)
	assert.Error(t, err)

	for i := 43; i < 129; i++ {
		info, err := New(uint(i))
		assert.NoError(t, err)

		assert.NotEmpty(t, info.CodeVerifier)
		assert.NotEmpty(t, info.CodeChallenge)

		assert.Equal(t, i, len(info.CodeVerifier))

		assert.Equal(t, info.CodeChallenge, ChallengeSHA256(info.CodeVerifier))

		decoded, err := base64.RawURLEncoding.DecodeString(info.Base64URLCodeVerifier())
		assert.NoError(t, err)
		assert.Equal(t, info.CodeVerifier, string(decoded))
	}

}
