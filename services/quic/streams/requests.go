/*
 * Copyright (C) 2025 The "MysteriumNetwork/node" Authors.
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

package streams

import (
	"context"
	"io"

	"github.com/rs/zerolog/log"
)

// ConnectStreams connect streams.
func ConnectStreams(c context.Context, downstream, upstream io.ReadWriteCloser, statsCallback func(string, uint64)) {
	uploadFinished := make(chan bool)
	downloadFinished := make(chan bool)

	go func() {
		copyStream(c, downstream, upstream, "Upload", statsCallback)
		close(uploadFinished)
	}()

	go func() {
		copyStream(c, upstream, downstream, "Download", statsCallback)
		close(downloadFinished)
	}()

	select {
	case <-downloadFinished:
	case <-uploadFinished:
	}

	if err := upstream.Close(); err != nil {
		log.Debug().Msg("Upstream close error: " + err.Error())
	}

	if err := downstream.Close(); err != nil {
		log.Debug().Msg("Downstream close error: " + err.Error())
	}
}
