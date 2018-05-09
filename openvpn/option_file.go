package openvpn

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
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

func (option optionFile) toCli() (string, error) {
	err := ioutil.WriteFile(option.filePath, []byte(option.content), 0600)
	if err != nil {
		return "", err
	}
	return "--" + option.name + " " + option.filePath, nil
}

func (option optionFile) toFile() (string, error) {
	escaped, err := escapeXmlTags(option.content)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("<%s>\n%s\n</%s>", option.name, escaped, option.name), nil
}

func escapeXmlTags(content string) (string, error) {
	var escaped bytes.Buffer
	err := xml.EscapeText(&escaped, []byte(content))
	if err != nil {
		return "", err
	}
	return escaped.String(), nil
}
