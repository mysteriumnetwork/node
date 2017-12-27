package identity

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVerifyFunctionReturnsTrueWhenSignatureIsCorrect(t *testing.T) {
	message := []byte("I am message!")
	signature := []byte{1, 2, 3}

	assert.True(t, NewVerifier().Verify(message, signature))
}

func TestVerifyFunctionReturnsFalseWhenSignatureIsIncorrect(t *testing.T) {
	message := []byte("I am message!")
	signature := []byte{1, 2, 3}

	assert.False(t, NewVerifier().Verify(message, signature))
}
