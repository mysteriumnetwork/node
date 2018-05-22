package management

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestAddress_String(t *testing.T) {
	addr := Address{"127.0.0.1", 8080}
	assert.Equal(t, "127.0.0.1:8080", addr.String())
}

func TestGetAddressFromString(t *testing.T) {
	addr := GetAddressFromString("127.0.0.1:8080")
	assert.Equal(t, "127.0.0.1", addr.IP)
	assert.Equal(t, 8080, addr.Port)
}
