package openvpn

import (
	"fmt"
)

func ConfigToArguments(config Config) ([]string, error) {
	arguments := make([]string, 0)

	for _, item := range config.options {
		option, ok := item.(optionCliSerializable)
		if !ok {
			return nil, fmt.Errorf("Unserializable option '%s': %#v", item.getName(), item)
		}

		err := option.toArguments(&arguments)
		if err != nil {
			return nil, err
		}
	}

	return arguments, nil
}

type optionCliSerializable interface {
	toArguments(arguments *[]string) error
}

func (option *optionFlag) toArguments(arguments *[]string) error {
	*arguments = append(*arguments, "--" + option.name)
	return nil
}

func (option *optionParam) toArguments(arguments *[]string) error {
	*arguments = append(*arguments, "--" + option.name, option.value)
	return nil
}

func (option *optionFile) toArguments(arguments *[]string) error {
	*arguments = append(*arguments, "--" + option.name, option.path)
	return nil
}