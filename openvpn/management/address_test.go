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
	addr, err := GetPortAndAddressFromString("127.0.0.1:8080")
	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1", addr.IP)
	assert.Equal(t, 8080, addr.Port)
}

func TestGetAddressFromStringFails(t *testing.T) {
	addr, err := GetPortAndAddressFromString("127.0.0.1::")
	assert.Equal(t, "Failed to parse port number.", err.Error())
	assert.Nil(t, addr)

	addr, err = GetPortAndAddressFromString("::")
	assert.Equal(t, "Failed to parse port number.", err.Error())
	assert.Nil(t, addr)

	addr, err = GetPortAndAddressFromString("127.0.0.1")
	assert.Equal(t, "Failed to parse address string.", err.Error())
	assert.Nil(t, addr)

	addr, err = GetPortAndAddressFromString("")
	assert.Equal(t, "Failed to parse address string.", err.Error())
	assert.Nil(t, addr)
}
