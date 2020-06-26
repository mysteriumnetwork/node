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

package svflags

import (
	"flag"

	"github.com/rs/zerolog"
)

// Supervisor CLI flags.
var (
	FlagVersion     = flag.Bool("version", false, "Print version")
	FlagInstall     = flag.Bool("install", false, "Install or repair myst supervisor")
	FlagUid         = flag.String("uid", "", "User ID for which supervisor socket should be installed (required)")
	FlagUninstall   = flag.Bool("uninstall", false, "Uninstall myst supervisor")
	FlagLogFilePath = flag.String("log-path", "", "Supervisor log file path")
	FlagLogLevel    = flag.String("log-level", zerolog.InfoLevel.String(), "Logging level")
	FlagWinService  = flag.Bool("winservice", false, "Run via service manager instead of standalone (windows only).")
)

// Parse parses supervisor flags.
func Parse() {
	flag.Parse()
}
