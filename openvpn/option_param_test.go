package openvpn

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestParam_Factory(t *testing.T) {
	option := OptionParam("very-value", "1234")
	assert.NotNil(t, option)
}

func TestParam_GetName(t *testing.T) {
	option := OptionParam("very-value", "1234")
	assert.Equal(t, "very-value", option.getName())
}

func TestParam_ToArguments(t *testing.T) {
	var arguments []string
	option := OptionParam("very-value", "1234")

	err := option.toArguments(&arguments)
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]string{"--very-value", "1234"},
		arguments,
	)
}

func TestParam_ToFile(t *testing.T) {
	option := OptionParam("very-value", "1234")

	optionValue, err := option.toFile()
	assert.NoError(t, err)
	assert.Equal(t, "very-value 1234", optionValue)
}