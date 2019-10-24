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

package logconfig

import (
	"fmt"
	"io"
	stdlog "log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/arthurkiller/rollingwriter"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	timestampFmt = "2006-01-02T15:04:05.000"
)

// Bootstrap configures logger defaults (console)
func Bootstrap() {
	var trimPrefixes = []string{
		"/vendor",
		"/go/pkg/mod",
	}
	cwd, _ := os.Getwd()
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		relFile := strings.TrimPrefix(file, cwd)
		for i := range trimPrefixes {
			relFile = trimLeftInclusive(relFile, trimPrefixes[i])
		}
		return fmt.Sprintf("%-41v", relFile+":"+strconv.Itoa(line))
	}

	logger := makeLogger(consoleWriter())
	setGlobalLogger(&logger)
}

// Configure configures logger using app config (console + file, level)
func Configure(opts *LogOptions) {
	CurrentLogOptions = *opts
	log.Info().Msgf("Log level: %s", opts.LogLevel)
	if opts.Filepath != "" {
		log.Info().Msg("Log file path: " + opts.Filepath)
		fileWriter, err := fileWriter(opts)
		if err != nil {
			log.Error().Err(err).Msg("Failed to configure file logger")
		} else {
			multiWriter := io.MultiWriter(consoleWriter(), fileWriter)
			logger := makeLogger(multiWriter)
			setGlobalLogger(&logger)
		}

	}
	log.Logger.Level(opts.LogLevel)
}

func consoleWriter() io.Writer {
	return zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: timestampFmt,
	}
}

func fileWriter(opts *LogOptions) (io.Writer, error) {
	cfg := &rollingwriter.Config{
		TimeTagFormat:      "2006.01.02",
		LogPath:            path.Dir(opts.Filepath),
		FileName:           path.Base(opts.Filepath),
		RollingPolicy:      rollingwriter.TimeRolling,
		RollingTimePattern: "0 0 0 * * *",
		WriterMode:         "lock",
	}
	fileWriter, err := rollingwriter.NewWriterFromConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to configure file logger")
	}
	return zerolog.ConsoleWriter{
		Out:        fileWriter,
		NoColor:    true,
		TimeFormat: timestampFmt,
	}, nil
}

func makeLogger(w io.Writer) zerolog.Logger {
	return log.Output(w).
		Level(zerolog.DebugLevel).
		With().
		Caller().
		Timestamp().
		Logger()
}

func setGlobalLogger(logger *zerolog.Logger) {
	log.Logger = *logger
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)
}

// trimLeftInclusive trims left pat of the string up to and including the prefix
func trimLeftInclusive(s string, prefix string) string {
	start := strings.Index(s, prefix)
	if start != -1 {
		return s[start+len(prefix):]
	}
	return s
}
