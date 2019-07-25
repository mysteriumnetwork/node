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

package version

import (
	"bytes"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/urfave/cli.v1"
)

func TestCommandRun(t *testing.T) {
	output := bytes.NewBufferString("")

	command := NewCommand("0.0.1-alpha-male")
	command.Run(cli.NewContext(
		&cli.App{Writer: output},
		flag.NewFlagSet("test", 0),
		nil,
	))

	assert.Equal(t, "0.0.1-alpha-male\n", output.String())
}
