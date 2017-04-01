package openvpn

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParam_Factory(t *testing.T) {
	option := OptionParam("very-value", "1234")
	assert.NotNil(t, option)
}

func TestParam_GetName(t *testing.T) {
	option := OptionParam("very-value", "1234")
	assert.Equal(t, "very-value", option.getName())
}

func TestParam_ToCli(t *testing.T) {
	option := OptionParam("very-value", "1234")

	optionValue, err := option.toCli()
	assert.NoError(t, err)
	assert.Equal(t, "--very-value 1234", optionValue)
}

func TestParam_ToFile(t *testing.T) {
	option := OptionParam("very-value", "1234")

	optionValue, err := option.toFile()
	assert.NoError(t, err)
	assert.Equal(t, "very-value 1234", optionValue)
}
