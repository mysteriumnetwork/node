/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package client

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// Command executes supervisor command.
func Command(args ...string) (result string, err error) {
	cmdLine := strings.Join(args, " ")
	log.Trace().Msgf("Supervisor command invoked: %q", cmdLine)
	conn, err := connect()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, err = fmt.Fprintln(conn, cmdLine)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	line := string(scanner.Bytes())
	parts := strings.SplitN(line, ": ", 2)
	status := parts[0]
	if status == "ok" {
		if len(parts) > 1 {
			result = parts[1]
		}
		return result, nil
	}

	message := strings.TrimPrefix(line, "error: ")
	return "", errors.New(message)
}
