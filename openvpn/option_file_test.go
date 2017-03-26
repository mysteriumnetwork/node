package openvpn

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestFile_Factory(t *testing.T) {
	option := OptionFile("special-file", "file.txt")
	assert.NotNil(t, option)
}

func TestFile_GetName(t *testing.T) {
	option := OptionFile("special-file", "file.txt")
	assert.Equal(t, "special-file", option.getName())
}

func TestFile_ToCli(t *testing.T) {
	option := OptionFile("special-file", "file.txt")

	optionValue, err := option.toCli()
	assert.NoError(t, err)
	assert.Equal(t, "--special-file file.txt", optionValue)
}

func TestFile_ToFileNotExisting(t *testing.T) {
	option := OptionFile("special-file", "file-notexisting.txt")

	optionValue, err := option.toFile()
	assert.Error(t, err)
	assert.EqualError(t, err, "open file-notexisting.txt: no such file or directory")
	assert.Empty(t, optionValue)
}

func TestFile_ToFile(t *testing.T) {
	option := OptionFile("special-file", "testdata/file.txt")

	optionValue, err := option.toFile()
	assert.NoError(t, err)
	assert.Equal(t, "<special-file>\n[filedata]\n</special-file>", optionValue)
}