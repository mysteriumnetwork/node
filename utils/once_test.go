package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProviderFunctionIsCalledOnce(t *testing.T) {
	val := 1

	incOnce := CallOnce(func() {
		val = val + 1
	})
	incOnce()
	incOnce()
	assert.Equal(t, 2, val)
}
