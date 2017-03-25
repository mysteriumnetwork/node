package openvpn

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestConfigToString(t *testing.T) {
	config := Config{}
	config.AddOptions(
		OptionFlag("enable-something"),
		OptionParam("very-value", "1234"),
	)

	output, err := ConfigToString(config)
	assert.Nil(t, err)
	assert.Equal(t, "enable-something\nvery-value 1234\n", output)
}