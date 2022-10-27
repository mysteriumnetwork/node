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

package proxyclient

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

const copyBufferSize = 128 * 1024

var bufferPool = NewBufferPool(copyBufferSize)

func proxyHTTP1(ctx context.Context, left, right net.Conn) {
	wg := sync.WaitGroup{}

	idleTimeout := 5 * time.Minute
	timeout := time.AfterFunc(idleTimeout, func() {
		left.Close()
		right.Close()
	})
	extend := func() {
		timeout.Reset(idleTimeout)
	}

	cpy := func(dst, src net.Conn) {
		defer wg.Done()

		copyBuffer(dst, src, extend)
		dst.Close()
	}
	wg.Add(2)
	go cpy(left, right)
	go cpy(right, left)
	groupDone := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		groupDone <- struct{}{}
	}()
	select {
	case <-ctx.Done():
		left.Close()
		right.Close()
	case <-groupDone:
		return
	}
	<-groupDone
	return
}

func proxyHTTP2(ctx context.Context, leftreader io.ReadCloser, leftwriter io.Writer, right net.Conn) {
	wg := sync.WaitGroup{}

	idleTimeout := 5 * time.Minute
	timeout := time.AfterFunc(idleTimeout, func() {
		leftreader.Close()
		right.Close()
	})
	extend := func() {
		timeout.Reset(idleTimeout)
	}

	ltr := func(dst net.Conn, src io.Reader) {
		defer wg.Done()
		copyBuffer(dst, src, extend)
		dst.Close()
	}
	rtl := func(dst io.Writer, src io.Reader) {
		defer wg.Done()
		copyBody(dst, src)
	}
	wg.Add(2)
	go ltr(right, leftreader)
	go rtl(leftwriter, right)
	groupDone := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		groupDone <- struct{}{}
	}()
	select {
	case <-ctx.Done():
		leftreader.Close()
		right.Close()
	case <-groupDone:
		return
	}
	<-groupDone
	return
}

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Connection",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func hijack(hijackable interface{}) (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := hijackable.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("connection does not support hijacking")
	}
	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, nil, err
	}
	var emptyTime time.Time
	err = conn.SetDeadline(emptyTime)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return conn, rw, nil
}

func flush(flusher interface{}) bool {
	f, ok := flusher.(http.Flusher)
	if !ok {
		return false
	}
	f.Flush()
	return true
}

func copyBody(wr io.Writer, body io.Reader) {
	buf := bufferPool.Get()
	defer bufferPool.Put(buf)

	for {
		bread, readErr := body.Read(buf)
		var writeErr error
		if bread > 0 {
			_, writeErr = wr.Write(buf[:bread])
			flush(wr)
		}
		if readErr != nil || writeErr != nil {
			break
		}
	}
}

func copyBuffer(dst io.Writer, src io.Reader, extend func()) (written int64, err error) {
	buf := bufferPool.Get()
	defer bufferPool.Put(buf)

	for {
		extend()
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errors.New("invalid write result")
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
