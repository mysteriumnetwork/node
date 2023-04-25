package pkce

import (
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

		assert.NotEmpty(t, info.codeVerifier)
		assert.NotEmpty(t, info.codeChallenge)

		assert.Equal(t, i, len(info.codeVerifier))

		assert.Equal(t, info.codeChallenge, ChallengeSHA256(info.codeVerifier))
	}

}
