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
	"fmt"
)

type optionStringSerializable interface {
	toFile() (string, error)
}

func ConfigToString(config Config) (string, error) {
	var output bytes.Buffer

	for _, item := range config.options {
		option, ok := item.(optionStringSerializable)
		if !ok {
			return "", fmt.Errorf("Unserializable option '%s': %#v", item.getName(), item)
		}

		optionValue, err := option.toFile()
		if err != nil {
			return "", err
		}
		fmt.Fprintln(&output, optionValue)
	}

	return string(output.Bytes()), nil
}
