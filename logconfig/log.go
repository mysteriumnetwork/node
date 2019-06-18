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
	"strings"

	"github.com/cihub/seelog"
)

// Logger provides a ceelog logger with prefix
type Logger struct {
	prefix string
}

// NewLogger provides a logger with prefix equal to package name of the caller
func NewLogger() *Logger {
	pkg := retrieveCallInfo().packageName
	localPkg := strings.Replace(pkg, "github.com/mysteriumnetwork/node/", "", 1)
	return &Logger{
		prefix: "[" + localPkg + "] ",
	}
}

// Tracef trace log fmt
func (l Logger) Tracef(format string, params ...interface{}) {
	seelog.Tracef(l.prefix+format, params...)
}

// Debugf debug log fmt
func (l Logger) Debugf(format string, params ...interface{}) {
	seelog.Debugf(l.prefix+format, params...)
}

// Infof info log fmt
func (l Logger) Infof(format string, params ...interface{}) {
	seelog.Infof(l.prefix+format, params...)
}

// Warnf warn log fmt
func (l Logger) Warnf(format string, params ...interface{}) error {
	return seelog.Warnf(l.prefix+format, params...)
}

// Errorf error log fmt
func (l Logger) Errorf(format string, params ...interface{}) error {
	return seelog.Errorf(l.prefix+format, params...)
}

// Criticalf critical log fmt
func (l Logger) Criticalf(format string, params ...interface{}) error {
	return seelog.Criticalf(l.prefix+format, params...)
}

// Trace trace log
func (l Logger) Trace(v ...interface{}) {
	seelog.Trace(append([]interface{}{l.prefix}, v...)...)
}

// Debug debug log
func (l Logger) Debug(v ...interface{}) {
	seelog.Debug(append([]interface{}{l.prefix}, v...)...)
}

// Info info log
func (l Logger) Info(v ...interface{}) {
	seelog.Info(append([]interface{}{l.prefix}, v...)...)
}

// Warn warn log
func (l Logger) Warn(v ...interface{}) error {
	return seelog.Warn(append([]interface{}{l.prefix}, v...)...)
}

// Error error log
func (l Logger) Error(v ...interface{}) error {
	return seelog.Error(append([]interface{}{l.prefix}, v...)...)
}

// Critical critical log
func (l Logger) Critical(v ...interface{}) error {
	return seelog.Critical(append([]interface{}{l.prefix}, v...)...)
}

// Flush flushes logs to output
func (l Logger) Flush() {
	seelog.Flush()
}
