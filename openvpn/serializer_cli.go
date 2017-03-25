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