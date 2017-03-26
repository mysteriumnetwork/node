package openvpn

import (
	"io/ioutil"
	"fmt"
)

func OptionFile(name, path string) optionFile {
	return optionFile{name, path}
}

type optionFile struct {
	name string
	path string
}

func (option optionFile) getName() string {
	return option.name
}

func (option optionFile) toCli() (string, error) {
	return "--" + option.name + " " + option.path, nil
}

func (option optionFile) toFile() (string, error) {
	fileContent, err := ioutil.ReadFile(option.path)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("<%s>\n%s\n</%s>", option.name, string(fileContent), option.name), nil
}