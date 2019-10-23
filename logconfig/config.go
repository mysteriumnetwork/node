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
	stdlog "log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cihub/seelog"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// TODO remove this after log rotation is re-done on zerolog
// leaving for the reference
//const seewayLogXMLConfigTemplate = `
//<seelog minlevel="{{.LogLevel}}">
//	<outputs formatid="main">
//		<console/>
//		{{ if (ne .Filepath "") }}
//		<rollingfile
//			formatid="main"
//			filename="{{.Filepath}}"
//			maxrolls="7"
//			type="date"
//			datepattern="2006.01.02"
//		/>
//		{{ end }}
//	</outputs>
//	<formats>
//		<format id="main" format="%UTCDate(2006-01-02T15:04:05.999999999) [%Level] %Msg%n"/>
//	</formats>
//</seelog>
//`

// Bootstrap loads log package into the overall system with debug defaults
func Bootstrap() {
	configureZerolog(&CurrentLogOptions)
}

// Configure loads log package into the overall system
func Configure(opts *LogOptions) {
	if opts != nil {
		CurrentLogOptions = *opts
	}
	configureZerolog(opts)
	log.Info().Msgf("Log level: %s", CurrentLogOptions.LogLevel)
	if CurrentLogOptions.Filepath != "" {
		log.Info().Msg("Log file path: " + CurrentLogOptions.Filepath)
	}
}

func configureZerolog(opts *LogOptions) {
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
	writer := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "2006-01-02T15:04:05.000",
	}
	log.Logger = log.Output(writer).
		Level(opts.LogLevel).
		With().
		Caller().
		Timestamp().
		Logger()
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)
	logger, err := seelog.LoggerFromWriterWithMinLevel(log.Logger, seelog.TraceLvl)
	if err != nil {
		log.Warn().Err(err).Msg("Could not configure seelog")
	}
	seelog.ReplaceLogger(logger)
}

// trimLeftInclusive trims left pat of the string up to and including the prefix
func trimLeftInclusive(s string, prefix string) string {
	start := strings.Index(s, prefix)
	if start != -1 {
		return s[start+len(prefix):]
	}
	return s
}
