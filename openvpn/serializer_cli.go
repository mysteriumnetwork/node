package openvpn

import (
	"fmt"
	"strings"
)

func (config *Config) ConfigToArguments() ([]string, error) {
	arguments := make([]string, 0)

	for _, item := range config.options {
		option, ok := item.(optionCliSerializable)
		if !ok {
			return nil, fmt.Errorf("Unserializable option '%s': %#v", item.getName(), item)
		}

		optionValue, err := option.toCli()
		if err != nil {
			return nil, err
		}

		optionArguments := strings.Split(optionValue, " ")
		arguments = append(arguments, optionArguments...)
	}

	return arguments, nil
}

type optionCliSerializable interface {
	toCli() (string, error)
}
