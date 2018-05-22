/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package bytescount

import (
	"github.com/mysterium/node/openvpn/management"
	"regexp"
	"strconv"
	"time"
)

// SessionStatsHandler is invoked when middleware receives statistics
type SessionStatsHandler func(SessionStats) error

const byteCountCommandTemplate = "bytecount %d"

type middleware struct {
	sessionStatsHandler SessionStatsHandler
	interval            time.Duration
}

// NewMiddleware returns new bytescount middleware
func NewMiddleware(sessionStatsHandler SessionStatsHandler, interval time.Duration) management.Middleware {
	return &middleware{
		sessionStatsHandler: sessionStatsHandler,
		interval:            interval,
	}
}

func (middleware *middleware) Start(commandWriter management.CommandWriter) error {
	_, err := commandWriter.SingleLineCommand(byteCountCommandTemplate, int(middleware.interval.Seconds()))
	return err
}

func (middleware *middleware) Stop(commandWriter management.CommandWriter) error {
	_, err := commandWriter.SingleLineCommand(byteCountCommandTemplate, 0)
	return err
}

func (middleware *middleware) ConsumeLine(line string) (consumed bool, err error) {
	rule, err := regexp.Compile("^>BYTECOUNT:(.*),(.*)$")
	if err != nil {
		return
	}

	match := rule.FindStringSubmatch(line)
	consumed = len(match) > 0
	if !consumed {
		return
	}

	bytesIn, err := strconv.Atoi(match[1])
	if err != nil {
		return
	}

	bytesOut, err := strconv.Atoi(match[2])
	if err != nil {
		return
	}

	stats := SessionStats{BytesSent: bytesOut, BytesReceived: bytesIn}
	err = middleware.sessionStatsHandler(stats)

	return
}
