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
	"github.com/cihub/seelog"
)

// CaptureLogs captures all logger output done during callback execution
func CaptureLogs(f func()) []string {
	logs := make([]string, 0)

	loggerOld := ReplaceLogger(NewLoggerCapture(&logs))
	defer ReplaceLogger(loggerOld)

	f()

	return logs
}

// NewLoggerCapture creates logger logs to given array of messages
func NewLoggerCapture(messages *[]string) seelog.LoggerInterface {
	logger, _ := seelog.LoggerFromCustomReceiver(
		&captureReceiver{messages},
	)
	return logger
}

type captureReceiver struct {
	Messages *[]string
}

func (receiver *captureReceiver) ReceiveMessage(message string, level seelog.LogLevel, context seelog.LogContextInterface) error {
	*receiver.Messages = append(*receiver.Messages, message)

	return nil
}

func (receiver *captureReceiver) AfterParse(initArgs seelog.CustomReceiverInitArgs) error {
	return nil
}
func (receiver *captureReceiver) Flush() {

}
func (receiver *captureReceiver) Close() error {
	return nil
}
