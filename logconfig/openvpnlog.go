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

	"github.com/rs/zerolog/log"
)

type zerologOpenvpnLogger struct {
}

// Error logs go-openvpn error.
func (l zerologOpenvpnLogger) Error(args ...interface{}) {
	strs := make([]string, len(args))
	for i, v := range args {
		strs[i] = fmt.Sprint(v)
	}
	log.Error().Msg(strings.Join(strs, " "))
}

// Warn logs go-openvpn warning.
func (l zerologOpenvpnLogger) Warn(args ...interface{}) {
	strs := make([]string, len(args))
	for i, v := range args {
		strs[i] = fmt.Sprint(v)
	}
	log.Warn().Msg(strings.Join(strs, " "))
}

// Info logs go-openvpn informational message.
func (l zerologOpenvpnLogger) Info(args ...interface{}) {
	strs := make([]string, len(args))
	for i, v := range args {
		strs[i] = fmt.Sprint(v)
	}
	log.Info().Msg(strings.Join(strs, " "))
}

// Debug logs go-openvpn debug message.
func (l zerologOpenvpnLogger) Debug(args ...interface{}) {
	strs := make([]string, len(args))
	for i, v := range args {
		strs[i] = fmt.Sprint(v)
	}
	log.Debug().Msg(strings.Join(strs, " "))
}
