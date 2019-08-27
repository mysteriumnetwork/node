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
	"strings"

	log "github.com/cihub/seelog"
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/urfave/cli.v1/altsrc"
)

// LogOptions log options
type LogOptions struct {
	logLevelInt log.LogLevel
	LogLevel    string
}

// CurrentLogOptions current log options
var CurrentLogOptions = LogOptions{
	logLevelInt: log.DebugLvl,
	LogLevel:    log.DebugStr,
}

var (
	logLevel = altsrc.NewStringFlag(cli.StringFlag{
		Name: "log-level",
		Usage: func() string {
			allLevels := []string{log.TraceStr, log.DebugStr, log.InfoStr, log.WarnStr, log.ErrorStr, log.CriticalStr, log.OffStr}
			return fmt.Sprintf("Set the logging level (%s)", strings.Join(allLevels, "|"))
		}(),
		Value: func() string {
			level := log.DebugStr
			return level
		}(),
	})
)

// RegisterFlags registers logger CLI flags
func RegisterFlags(flags *[]cli.Flag) {
	*flags = append(*flags, logLevel)
}

// ParseFlags parses logger CLI flags from context
func ParseFlags(ctx *cli.Context) LogOptions {
	level := ctx.GlobalString("log-level")
	levelInt, found := log.LogLevelFromString(level)
	if !found {
		levelInt = log.DebugLvl
		level = log.DebugStr
	}
	return LogOptions{
		logLevelInt: levelInt,
		LogLevel:    level,
	}
}
