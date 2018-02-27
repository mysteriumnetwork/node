package openvpn

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestErrorIsReturnedOnBadBinaryPath(t *testing.T) {
	assert.Error(t, CheckOpenvpnBinary("non-existent-binary"))
}

func TestErrorIsReturnedOnExitCodeZero(t *testing.T) {
	assert.Error(t, CheckOpenvpnBinary("testdata/exit-with-zero.sh"))
}

func TestNoErrorIsReturnedOnExitCodeOne(t *testing.T) {
	assert.NoError(t, CheckOpenvpnBinary("testdata/exit-with-one.sh"))
}
