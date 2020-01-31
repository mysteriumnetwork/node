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
	"os/exec"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Exec activates given args
var Exec = func(args []string) ([]string, error) {
	args = append([]string{"sudo", "/usr/sbin/ipset"}, args...)

	log.Debug().Msgf("[cmd] %v", args)
	output, err := exec.Command("sudo", args...).CombinedOutput()
	if err != nil {
		log.Debug().Err(err).Msgf("[cmd error] %v output: %v", args, string(output))
		return nil, errors.Wrap(err, "ipset cmd error")
	}

	outputScanner := bufio.NewScanner(bytes.NewBuffer(output))
	var lines []string
	for outputScanner.Scan() {
		lines = append(lines, outputScanner.Text())
	}
	return lines, outputScanner.Err()
}
