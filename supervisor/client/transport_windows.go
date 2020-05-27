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
	"fmt"
	"io"
	"time"

	"github.com/Microsoft/go-winio"
)

const sock = `\\.\pipe\mystpipe`

func connect() (io.ReadWriteCloser, error) {
	timeout := 5 * time.Second
	conn, err := winio.DialPipe(sock, &timeout)
	if err != nil {
		return nil, fmt.Errorf("could not connect to the supervisor socket %s: %w", sock, err)
	}
	return conn, nil
}
