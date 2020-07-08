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

package daemon

import (
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog/log"
)

type responder struct {
	io.Writer
}

func (r *responder) ok(result ...string) {
	args := []string{"ok"}
	args = append(args, result...)
	r.message(strings.Join(args, ": "))
}

func (r *responder) err(result ...error) {
	args := []string{"error"}
	for _, err := range result {
		args = append(args, err.Error())
	}
	r.message(strings.Join(args, ": "))
}

func (r *responder) message(msg string) {
	log.Debug().Msgf("< %s", msg)
	if _, err := fmt.Fprintln(r, msg); err != nil {
		log.Err(err).Msgf("Could not send message: %q", msg)
	}
}
