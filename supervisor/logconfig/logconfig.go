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

package logconfig

import (
	"fmt"
	stdlog "log"

	"github.com/mysteriumnetwork/node/logconfig/rollingwriter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	timestampFmt = "2006-01-02T15:04:05.000"
)

// Configure configures supervisor global logger instance.
func Configure(logPath string) error {
	rw, err := rollingwriter.NewRollingWriter(logPath)
	if err != nil {
		return fmt.Errorf("could not to create rolling logs writer: %w", err)
	}
	if err := rw.CleanObsoleteLogs(); err != nil {
		log.Printf("Failed to cleanup obsolete logs: %v", err)
	}

	logger := log.Output(zerolog.ConsoleWriter{Out: rw, NoColor: true, TimeFormat: timestampFmt}).
		Level(zerolog.DebugLevel).
		With().
		Timestamp().
		Logger()

	log.Logger = logger
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	return nil
}
