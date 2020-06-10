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
	"io"
	stdlog "log"
	"os"

	"github.com/mysteriumnetwork/node/logconfig/rollingwriter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	timestampFmt = "2006-01-02T15:04:05.000"
)

// LogOptions describes logging options.
type LogOptions struct {
	LogLevel string
	Filepath string
}

// Configure configures supervisor global logger instance.
func Configure(opts LogOptions) error {
	logLevel, err := zerolog.ParseLevel(opts.LogLevel)
	if err != nil {
		return fmt.Errorf("could not parse log level: %w", err)
	}
	log.Logger = log.Logger.Level(logLevel)

	// Set default file path if not specified.
	if opts.Filepath == "" {
		var err error
		opts.Filepath, err = defaultLogPath()
		if err != nil {
			return fmt.Errorf("could not get default log path: %w", err)
		}
	}

	logsWriter, err := newLogWriter(opts.Filepath)
	if err != nil {
		return fmt.Errorf("could not create logs writer: %w", err)
	}

	logger := log.Output(logsWriter).
		Level(logLevel).
		With().
		Timestamp().
		Logger()

	log.Logger = logger
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	return nil
}

// newLogWriter returns multi writer which can log to both file and console.
func newLogWriter(filepath string) (io.Writer, error) {
	rw, err := rollingwriter.NewRollingWriter(filepath)
	if err != nil {
		return nil, fmt.Errorf("could not to create rolling logs writer: %w", err)
	}

	if err := rw.CleanObsoleteLogs(); err != nil {
		log.Err(err).Msg("Failed to cleanup obsolete logs")
	}

	fileWriter := zerolog.ConsoleWriter{
		Out:        rw,
		NoColor:    true,
		TimeFormat: timestampFmt,
	}
	stderrWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		NoColor:    true,
		TimeFormat: timestampFmt,
	}
	return io.MultiWriter(fileWriter, stderrWriter), nil
}
