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

package logconfig

import (
	"fmt"
	"path"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/urfave/cli.v1/altsrc"
)

// LogOptions log options
type LogOptions struct {
	LogLevel zerolog.Level
	LogHTTP  bool
	Filepath string
}

// CurrentLogOptions current log options
var CurrentLogOptions = LogOptions{
	LogLevel: zerolog.DebugLevel,
	LogHTTP:  false,
}

var (
	logLevel = altsrc.NewStringFlag(cli.StringFlag{
		Name: "log-level",
		Usage: func() string {
			allLevels := []string{
				zerolog.DebugLevel.String(),
				zerolog.InfoLevel.String(),
				zerolog.WarnLevel.String(),
				zerolog.FatalLevel.String(),
				zerolog.PanicLevel.String(),
				zerolog.Disabled.String(),
			}
			return fmt.Sprintf("Set the logging level (%s)", strings.Join(allLevels, "|"))
		}(),
		Value: zerolog.DebugLevel.String(),
	})
	logHttp = altsrc.NewBoolFlag(cli.BoolFlag{
		Name:  "log.http",
		Usage: "Enable HTTP payload logging",
	})
)

// RegisterFlags registers logger CLI flags
func RegisterFlags(flags *[]cli.Flag) {
	*flags = append(*flags, logLevel, logHttp)
}

// ParseFlags parses logger CLI flags from context
func ParseFlags(ctx *cli.Context, logDir string) LogOptions {
	level, err := zerolog.ParseLevel(ctx.GlobalString(logLevel.Name))
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse logging level")
		level = zerolog.DebugLevel
	}
	return LogOptions{
		LogLevel: level,
		Filepath: path.Join(logDir, "mysterium-node"),
		LogHTTP:  ctx.GlobalBool(logHttp.Name),
	}
}
