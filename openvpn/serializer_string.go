package openvpn

import (
	"fmt"
	"bytes"
	"io/ioutil"
)

func ConfigToString(config Config) (string, error) {
	var output bytes.Buffer

	for _, item := range config.options {
		option, ok := item.(optionStringSerializable)
		if !ok {
			return "", fmt.Errorf("Unserializable option '%s': %#v", item.getName(), item)
		}

		optionValue, err := option.toString()
		if err != nil {
			return "", err
		}
		fmt.Fprintln(&output, optionValue)
	}

	return string(output.Bytes()), nil
}

type optionStringSerializable interface {
	toString() (string, error)
}

func (option optionFlag) toString() (string, error) {
	return option.name, nil
}

func (option optionParam) toString() (string, error) {
	return option.name + " " + option.value, nil
}

func (option optionFile) toString() (string, error) {
	fileContent, err := ioutil.ReadFile(option.path)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("<%s>%s</%s>", option.name, string(fileContent), option.name), nil
}