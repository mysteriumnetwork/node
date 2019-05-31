package ip

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalhostOutboundIPIsReturned(t *testing.T) {
	checkAddress = "localhost:5555"
	ip, err := GetOutbound()
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1", ip.String())
}
