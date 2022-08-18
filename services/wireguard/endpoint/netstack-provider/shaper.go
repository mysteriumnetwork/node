/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package netstack_provider

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

type Reader struct {
	r       io.Reader
	limiter *rate.Limiter
	ctx     context.Context
}

// NewReader returns a reader that implements io.Reader with rate limiting.
func NewReader(r io.Reader, limiter *rate.Limiter) *Reader {
	return &Reader{
		r:       r,
		limiter: limiter,
		ctx:     context.Background(),
	}
}

// NewReaderWithContext returns a reader that implements io.Reader with rate limiting.
func NewReaderWithContext(r io.Reader, ctx context.Context) *Reader {
	return &Reader{
		r:   r,
		ctx: ctx,
	}
}

// Read reads bytes into p.
func (s *Reader) Read(p []byte) (int, error) {
	if s.limiter == nil {
		return s.r.Read(p)
	}
	n, err := s.r.Read(p)
	if err != nil {
		return n, err
	}
	if err := s.limiter.WaitN(s.ctx, n); err != nil {
		return n, err
	}
	return n, nil
}
