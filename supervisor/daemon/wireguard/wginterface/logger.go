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

package wginterface

import (
	"io"
	"io/ioutil"
	stdlog "log"

	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/device"
)

// newLogger creates WireGuard logger which uses already configured global zero log instance.
func newLogger(level int, prepend string) *device.Logger {
	output := log.Logger
	logger := new(device.Logger)

	logErr, logInfo, logDebug := func() (io.Writer, io.Writer, io.Writer) {
		if level >= device.LogLevelDebug {
			return output, output, output
		}
		if level >= device.LogLevelInfo {
			return output, output, ioutil.Discard
		}
		if level >= device.LogLevelError {
			return output, ioutil.Discard, ioutil.Discard
		}
		return ioutil.Discard, ioutil.Discard, ioutil.Discard
	}()

	logger.Debug = stdlog.New(logDebug,
		"DEBUG: "+prepend,
		stdlog.Ldate|stdlog.Ltime,
	)

	logger.Info = stdlog.New(logInfo,
		"INFO: "+prepend,
		stdlog.Ldate|stdlog.Ltime,
	)
	logger.Error = stdlog.New(logErr,
		"ERROR: "+prepend,
		stdlog.Ldate|stdlog.Ltime,
	)
	return logger
}
