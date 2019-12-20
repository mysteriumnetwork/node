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
	"strconv"
	"strings"
	"time"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	timestampFmt = "2006-01-02T15:04:05.000"
)

// Bootstrap configures logger defaults (console).
func Bootstrap() {
	var trimPrefixes = []string{
		"/github.com/mysteriumnetwork/node",
		"/vendor",
		"/go/pkg/mod",
	}
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		var ok bool
		for _, prefix := range trimPrefixes {
			file, ok = trimLeftInclusive(file, prefix)
			if ok {
				break
			}
		}
		return fmt.Sprintf("%-41v", file+":"+strconv.Itoa(line))
	}

	openvpn.UseLogger(zerologOpenvpnLogger{})
	logger := makeLogger(consoleWriter())
	setGlobalLogger(&logger)
}

// Configure configures logger using app config (console + file, level).
func Configure(opts *LogOptions) {
	CurrentLogOptions = *opts
	log.Info().Msgf("Log level: %s", opts.LogLevel)
	if opts.Filepath != "" {
		log.Info().Msgf("Log file path: %s", opts.Filepath)
		rollingWriter, err := newRollingWriter(opts)
		if err != nil {
			log.Err(err).Msg("Failed to configure file logger")
		} else {
			multiWriter := io.MultiWriter(consoleWriter(), rollingWriter.zeroLogger())
			logger := makeLogger(multiWriter)
			setGlobalLogger(&logger)
		}
		if err := rollingWriter.cleanObsoleteLogs(); err != nil {
			log.Err(err).Msg("Failed to cleanup obsolete logs")
		}
	}
	log.Logger = log.Logger.Level(opts.LogLevel)
}

func consoleWriter() io.Writer {
	return zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: timestampFmt,
	}
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

// trimLeftInclusive trims left part of the string up to and including the prefix.
func trimLeftInclusive(s string, prefix string) (string, bool) {
	start := strings.Index(s, prefix)
	if start != -1 {
		return s[start+len(prefix):], true
	}
	return s, false
}
