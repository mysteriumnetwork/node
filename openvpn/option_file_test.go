package openvpn

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestFile_Factory(t *testing.T) {
	option := OptionFile("special-file", "", "file.txt")
	assert.NotNil(t, option)
}

func TestFile_GetName(t *testing.T) {
	option := OptionFile("special-file", "", "file.txt")
	assert.Equal(t, "special-file", option.getName())
}

func TestFile_ToCli(t *testing.T) {
	filePath := "testdataoutput/file.txt"
	fileContent := "file-content"

	option := OptionFile("special-file", fileContent, filePath)

	optionValue, err := option.toCli()
	assert.NoError(t, err)
	assert.Equal(t, "--special-file testdataoutput/file.txt", optionValue)
	readedContent, err := ioutil.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, fileContent, string(readedContent))
}

func TestFile_ToCliNotExistingDir(t *testing.T) {
	option := OptionFile("special-file", "file-content", "nodir/file.txt")

	optionValue, err := option.toCli()
	assert.Error(t, err)
	assert.EqualError(t, err, "open nodir/file.txt: no such file or directory")
	assert.Empty(t, optionValue)
}

func TestFile_ToFile(t *testing.T) {
	option := OptionFile("special-file", "[filedata]", "testdata/file.txt")

	optionValue, err := option.toFile()
	assert.NoError(t, err)
	assert.Equal(t, "<special-file>\n[filedata]\n</special-file>", optionValue)
}
