package openvpn

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestFlag_Factory(t *testing.T) {
	option := OptionFlag("enable-something")
	assert.NotNil(t, option)
}

func TestFlag_GetName(t *testing.T) {
	option := OptionFlag("enable-something")
	assert.Equal(t, "enable-something", option.getName())
}

func TestFlag_ToArguments(t *testing.T) {
	var arguments []string
	option := OptionFlag("enable-something")

	err := option.toArguments(&arguments)
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]string{"--enable-something"},
		arguments,
	)
}

func TestFlag_ToFile(t *testing.T) {
	option := OptionFlag("enable-something")

	optionValue, err := option.toFile()
	assert.NoError(t, err)
	assert.Equal(t, "enable-something", optionValue)
}