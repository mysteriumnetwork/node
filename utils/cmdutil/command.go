/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package cmdutil

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// SudoExec executes external command with sudo privileges and logs output on the debug level.
// It returns a combined stderr and stdout output and exit code in case of an error.
func SudoExec(args ...string) error {
	args = append([]string{"sudo"}, args...)
	return Exec(args...)
}

// Exec executes external command and logs output on the debug level.
// It returns a combined stderr and stdout output and exit code in case of an error.
func Exec(args ...string) error {
	_, err := ExecOutput(args...)
	return err
}

// ExecOutput executes external command and logs output on the debug level.
// It returns a combined stderr and stdout output and exit code in case of an error.
func ExecOutput(args ...string) (output string, err error) {
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	logSkipFrame := log.With().CallerWithSkipFrameCount(3).Logger()
	(&logSkipFrame).Debug().Msgf("%q output:\n%s", strings.Join(args, " "), out)
	if err != nil {
		return string(out), errors.Errorf("%q: %v output: %s", strings.Join(args, " "), err, out)
	}
	return string(out), nil
}
