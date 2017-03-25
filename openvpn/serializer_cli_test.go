package openvpn

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestConfigToArguments(t *testing.T) {
	config := Config{}
	config.AddOptions(
		OptionFlag("enable-something"),
		OptionParam("very-value", "1234"),
	)

	arguments, err := ConfigToArguments(config)
	assert.Nil(t, err)
	assert.Equal(t,
		[]string{
			"--enable-something",
			"--very-value", "1234",
		},
		arguments,
	)
}