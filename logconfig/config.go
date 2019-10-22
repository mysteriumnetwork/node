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

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

const rootPkg = "github.com/mysteriumnetwork/node"

func configureZerolog(opts *LogOptions) {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		pkgStart := strings.Index(file, rootPkg)
		var relFile string
		if pkgStart > 0 {
			relFile = file[pkgStart+len(rootPkg):]
			relFile = strings.TrimPrefix(relFile, "/vendor")
		} else {
			relFile = file
		}
		caller := relFile + ":" + strconv.Itoa(line)
		return padRight(caller, 35)
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
