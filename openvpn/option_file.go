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

package openvpn

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"strings"
)

func OptionFile(name, content string, filePath string) optionFile {
	return optionFile{name, content, filePath}
}

type optionFile struct {
	name     string
	content  string
	filePath string
}

func (option optionFile) getName() string {
	return option.name
}

func (option optionFile) toCli() ([]string, error) {
	err := ioutil.WriteFile(option.filePath, []byte(option.content), 0600)
	if err != nil {
		return nil, err
	}
	return []string{"--" + option.name, option.filePath}, nil
}

func (option optionFile) toFile() (string, error) {
	escaped, err := escapeXmlTags(option.content)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("<%s>\n%s\n</%s>", option.name, escaped, option.name), nil
}

func escapeXmlTags(content string) (string, error) {
	var buff bytes.Buffer
	//escapes xml tags...
	err := xml.EscapeText(&buff, []byte(content))
	if err != nil {
		return "", err
	}
	//...but does too much - also escapes new lines and breaks PEM format - undo that
	escaped := strings.Replace(buff.String(), "&#xA;", "\n", -1)

	return escaped, nil
}
