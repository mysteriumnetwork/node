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
	"io"
	"path"

	"github.com/arthurkiller/rollingwriter"
	"github.com/rs/zerolog"
)

type rollingWriter struct {
	config rollingwriter.Config
	fw     io.Writer
}

func newRollingWriter(opts *LogOptions) (writer *rollingWriter, err error) {
	writer = &rollingWriter{}
	writer.config = rollingwriter.Config{
		TimeTagFormat:      "2006.01.02",
		LogPath:            path.Dir(opts.Filepath),
		FileName:           path.Base(opts.Filepath),
		RollingPolicy:      rollingwriter.TimeRolling,
		RollingTimePattern: "0 0 0 * * *",
		WriterMode:         "lock",
	}
	writer.fw, err = rollingwriter.NewWriterFromConfig(&writer.config)
	return writer, err
}

func (w *rollingWriter) zeroLogger() io.Writer {
	return zerolog.ConsoleWriter{
		Out:        w.fw,
		NoColor:    true,
		TimeFormat: timestampFmt,
	}
}
