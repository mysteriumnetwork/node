/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package cli

import (
	"errors"
	"strings"
)

func parseCommandAndArgs(args string) (string, []string, error) {
	quoted := false
	fields := strings.FieldsFunc(args, func(r rune) bool {
		if r == '"' || r == '\'' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})

	switch len(fields) {
	case 0:
		return "", nil, errors.New("no command provided")
	case 1:
		return fields[0], []string{}, nil
	default:
		cmd := fields[0]
		cmdArgs := fields[1:]
		clean := make([]string, 0)

		for _, v := range cmdArgs {
			switch {
			case strings.Contains(v, "'"):
				v = strings.ReplaceAll(v, "'", "")
			case strings.Contains(v, "\""):
				v = strings.ReplaceAll(v, "\"", "")
			}
			clean = append(clean, v)
		}

		return cmd, clean, nil
	}
}

func validateArgs(args string) error {
	countSingle := strings.Count(args, "'")
	countDouble := strings.Count(args, "\"")

	if countSingle%2 != 0 || countDouble%2 != 0 {
		return errors.New("wrong format command strings starting with ' or \" should be closed")
	}

	return nil
}
