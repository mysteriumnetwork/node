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
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/urfave/cli"
)

type options struct {
	LogLevel string
}

var opts = options{
	LogLevel: log.DebugStr,
}

var (
	logLevel = cli.StringFlag{
		Name: "log-level, l",
		Usage: func() string {
			allLevels := []string{log.TraceStr, log.DebugStr, log.InfoStr, log.WarnStr, log.ErrorStr, log.CriticalStr, log.OffStr}
			return fmt.Sprintf("Set the logging level (%s)", strings.Join(allLevels, "|"))
		}(),
		Value: func() string {
			level := log.DebugStr
			if metadata.VersionAsString() == "source.dev-build" {
				level = log.TraceStr
			}
			return level
		}(),
	}
)

// RegisterFlags registers logger CLI flags
func RegisterFlags(flags *[]cli.Flag) {
	*flags = append(*flags, logLevel)
}

// ParseFlags parses logger CLI flags from context
func ParseFlags(ctx *cli.Context) {
	opts = options{ctx.GlobalString("log-level")}
}
