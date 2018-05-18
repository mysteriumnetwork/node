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

package auth

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

func parseClientEvent(line string) (clientEventType, string, error) {
	rule, err := regexp.Compile("^(\\w+),(.*)$")
	if err != nil {
		return "", "", err
	}
	match := rule.FindStringSubmatch(line)
	if len(match) < 3 {
		return "", "", errors.New("unable to parse event: " + line)
	}
	event := clientEventType(match[1])
	return event, match[2], nil
}

func parseEnvVar(data string) (string, string, error) {
	slice := strings.SplitN(data, "=", 2)
	if len(slice) == 2 {
		return slice[0], slice[1], nil
	} else if len(slice) == 1 {
		return slice[0], "", nil
	}
	return "", "", errors.New("invalid env var: " + data)
}

func parseIDAndKey(data string) (int, int, error) {
	rule, err := regexp.Compile("^(\\d+),(\\d+)$")
	if err != nil {
		return undefined, undefined, err
	}
	match := rule.FindStringSubmatch(data)
	if len(match) < 3 {
		return undefined, undefined, errors.New("unable to parse identifiers: " + data)
	}
	ID, err := strconv.Atoi(match[1])
	if err != nil {
		return undefined, undefined, err
	}
	key, err := strconv.Atoi(match[2])
	if err != nil {
		return undefined, undefined, err
	}
	return ID, key, nil
}

func parseID(data string) (int, error) {
	rule, err := regexp.Compile("^(\\d+)$")
	if err != nil {
		return undefined, err
	}
	match := rule.FindStringSubmatch(data)
	if len(match) < 2 {
		return undefined, errors.New("unable to parse identifier: " + data)
	}
	ID, err := strconv.Atoi(match[1])
	if err != nil {
		return undefined, err
	}
	return ID, nil
}
