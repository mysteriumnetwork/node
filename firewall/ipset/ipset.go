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

package ipset

import (
	"bufio"
	"bytes"

	"github.com/mysteriumnetwork/node/utils/cmdutil"
	"github.com/pkg/errors"
)

// Exec activates given args
var Exec = defaultExec

func defaultExec(args []string) ([]string, error) {
	args = append([]string{"sudo", "/usr/sbin/ipset"}, args...)
	output, err := cmdutil.ExecOutput(args...)
	if err != nil {
		return nil, errors.Wrap(err, "ipset cmd error")
	}

	outputScanner := bufio.NewScanner(bytes.NewBufferString(output))
	var lines []string
	for outputScanner.Scan() {
		lines = append(lines, outputScanner.Text())
	}
	return lines, outputScanner.Err()
}
