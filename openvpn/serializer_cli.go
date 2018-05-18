/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
