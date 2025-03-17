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
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
)

func copyStream(ctx context.Context, src io.Reader, dst io.Writer, desc string, statsCallback func(string, uint64)) {
	n, err := copyBuffer(ctx, dst, src, desc, statsCallback)
	if err != nil {
		log.Trace().Err(err).Msgf("Failed to transfer: %s,  %d", desc, n)
		return
	}
}

func copyBuffer(ctx context.Context, dst io.Writer, src io.Reader, desc string, statsCallback func(string, uint64)) (int64, error) {
	var buf []byte
	var err error

	written := int64(0)

	size := 32 * 1024
	if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
		if l.N < 1 {
			size = 1
		} else {
			size = int(l.N)
		}
	}
	buf = make([]byte, size)

	for {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = fmt.Errorf("invalid write count")
				}
			}

			if statsCallback != nil {
				statsCallback(desc, uint64(nw))
			}

			written += int64(nw)
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, fmt.Errorf("read/write count mismatch")
			}
		}

		if er != nil {
			if er != io.EOF {
				err = er
			}
			return written, err
		}
	}
}
