/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogCapturer captures logging messages to an in-memory slice
type LogCapturer struct {
	logs     []string
	original zerolog.Logger
}

// NewLogCapturer creates a LogCapturer
func NewLogCapturer() *LogCapturer {
	return &LogCapturer{logs: []string{}}
}

// Attach attaches LogCapturer hook to the global zerolog instance
func (l *LogCapturer) Attach() {
	l.original = log.Logger
	log.Logger = log.Logger.Hook(l)
}

// Detach restores original global zerolog instance
func (l *LogCapturer) Detach() {
	log.Logger = l.original
}

// Run stores log message for later access (zerolog hook)
func (l *LogCapturer) Run(e *zerolog.Event, level zerolog.Level, message string) {
	l.logs = append(l.logs, message)
}

// Messages returns all captures log messages
func (l *LogCapturer) Messages() []string {
	return l.logs
}
