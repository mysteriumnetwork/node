package openvpn

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestConfigToArguments(t *testing.T) {
	config := Config{}
	config.AddOptions(
		OptionFlag("flag"),
		OptionFlag("spacy flag"),
		OptionParam("value", "1234"),
		OptionParam("very-value", "1234 5678"),
		OptionParam("spacy value", "1234 5678"),
	)

	arguments, err := ConfigToArguments(config)
	assert.Nil(t, err)
	assert.Equal(t,
		[]string{
			"--flag",
			"--spacy", "flag",
			"--value", "1234",
			"--very-value", "1234", "5678",
			"--spacy", "value", "1234", "5678",
		},
		arguments,
	)
}