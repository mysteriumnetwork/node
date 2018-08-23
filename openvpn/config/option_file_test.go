/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
	filename := filepath.Join("testdataoutput", "file.txt")
	os.Remove(filename)
	fileContent := "file-content"

	option := OptionFile("special-file", fileContent, filename)

	optionValue, err := option.toCli()
	assert.NoError(t, err)
	assert.Equal(t, []string{"--special-file", filename}, optionValue)
	readedContent, err := ioutil.ReadFile(filename)
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
	option := OptionFile("special-file", "[filedata]", "not-important")

	optionValue, err := option.toFile()
	assert.NoError(t, err)
	assert.Equal(t, "<special-file>\n[filedata]\n</special-file>", optionValue)
}

func TestFile_ToFileXmlTagsAreEscaped(t *testing.T) {
	option := OptionFile("file-name", "</file-name>This param is injected!\nNew line", "not-important")

	optionValue, err := option.toFile()
	assert.NoError(t, err)
	assert.Equal(
		t,
		`<file-name>
&lt;/file-name&gt;This param is injected!
New line
</file-name>`,
		optionValue,
	)
}
