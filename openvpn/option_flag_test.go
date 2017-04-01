package openvpn

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFlag_Factory(t *testing.T) {
	option := OptionFlag("enable-something")
	assert.NotNil(t, option)
}

func TestFlag_GetName(t *testing.T) {
	option := OptionFlag("enable-something")
	assert.Equal(t, "enable-something", option.getName())
}

func TestFlag_ToCli(t *testing.T) {
	option := OptionFlag("enable-something")

	optionValue, err := option.toCli()
	assert.NoError(t, err)
	assert.Equal(t, "--enable-something", optionValue)
}

func TestFlag_ToFile(t *testing.T) {
	option := OptionFlag("enable-something")

	optionValue, err := option.toFile()
	assert.NoError(t, err)
	assert.Equal(t, "enable-something", optionValue)
}
