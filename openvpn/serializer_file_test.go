package openvpn

import (
	"github.com/stretchr/testify/assert"
	"testing"
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
