package shaper

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
		r:   r,
		limiter: limiter,
		ctx: context.Background(),
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
