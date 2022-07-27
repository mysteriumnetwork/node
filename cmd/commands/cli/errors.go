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
	"fmt"
	"strings"

	"github.com/mysteriumnetwork/go-rest/apierror"
)

var (
	errWrongArgumentCount = errors.New("wrong number of arguments")
	errUnknownArgument    = errors.New("unknown argument")
	errTimeout            = errors.New("operation timed out")
)

func errUnknownSubCommand(cmd string) error {
	return fmt.Errorf("unknown sub-command '%s'", cmd)
}

func formatForHuman(err error) string {
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Detail()
	}
	msg := err.Error()
	if len(msg) < 1 {
		return msg
	}
	return strings.ToUpper(string(msg[0])) + msg[1:]
}
