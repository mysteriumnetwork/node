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
		prefix: fmt.Sprintf("[%s] ", localPkg),
	}
}

// NewNamespaceLogger provides a package name and a namespace prefix.
// Should be used if there is a need for more than one logger in the package.
func NewNamespaceLogger(ns string) *Logger {
	pkg := retrieveCallInfo().packageName
	localPkg := strings.Replace(pkg, "github.com/mysteriumnetwork/node/", "", 1)
	return &Logger{
		prefix: fmt.Sprintf("[%s:%s] ", localPkg, ns),
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
func (l Logger) Warnf(format string, params ...interface{}) {
	_ = seelog.Warnf(l.prefix+format, params...)
}

// Errorf error log fmt
func (l Logger) Errorf(format string, params ...interface{}) {
	_ = seelog.Errorf(l.prefix+format, params...)
}

// Criticalf critical log fmt
func (l Logger) Criticalf(format string, params ...interface{}) {
	_ = seelog.Criticalf(l.prefix+format, params...)
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
func (l Logger) Warn(v ...interface{}) {
	_ = seelog.Warn(append([]interface{}{l.prefix}, v...)...)
}

// Error error log
func (l Logger) Error(v ...interface{}) {
	_ = seelog.Error(append([]interface{}{l.prefix}, v...)...)
}

// Critical critical log
func (l Logger) Critical(v ...interface{}) {
	_ = seelog.Critical(append([]interface{}{l.prefix}, v...)...)
}

// IsTrace indicates if trace should be logged
func (Logger) IsTrace() bool {
	return CurrentLogOptions.logLevelInt <= seelog.TraceLvl
}

// Flush flushes logs to output
func (l Logger) Flush() {
	seelog.Flush()
}
