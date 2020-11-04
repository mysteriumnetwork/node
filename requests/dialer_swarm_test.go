/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package requests

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_DialerSwarm_UsesDefaultResolver(t *testing.T) {
	// given
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// when
	dialer := NewDialerSwarm("127.0.0.1")
	conn, err := dialer.DialContext(context.Background(), ln.Addr().Network(), ln.Addr().String())

	// then
	assert.NotNil(t, conn)
	assert.NoError(t, err)
}

func Test_DialerSwarm_CustomResolverSuccessfully(t *testing.T) {
	// given
	ln, err := net.Listen("tcp", "127.0.0.1:12345")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	dialer := NewDialerSwarm("127.0.0.1")
	dialer.ResolveContext = func(_ context.Context, _, host string) ([]string, error) {
		if host == "dns-is-faked.golang" {
			return []string{"127.0.0.1", "2001:db8::a3"}, nil
		}

		return nil, &net.DNSError{Err: "unmapped address", Name: host, IsNotFound: true}
	}

	// when
	conn, err := dialer.DialContext(context.Background(), "tcp", "dns-is-faked.golang:12345")

	// then
	assert.NotNil(t, conn)
	assert.NoError(t, err)
}

func Test_DialerSwarm_CustomResolverWithSomeUnreachableIPs(t *testing.T) {
	// given
	ln, err := net.Listen("tcp", "127.0.0.1:12345")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	dialer := NewDialerSwarm("127.0.0.1")
	dialer.ResolveContext = func(_ context.Context, _, host string) ([]string, error) {
		return []string{"2001:db8::a3", "127.0.0.1"}, nil
	}

	// when
	conn, err := dialer.DialContext(context.Background(), "tcp", "dns-is-faked.golang:12345")

	// then
	assert.NotNil(t, conn)
	assert.NoError(t, err)
}

func Test_DialerSwarm_CustomResolverWithAllUnreachableIPs(t *testing.T) {
	dialer := NewDialerSwarm("127.0.0.1")
	dialer.ResolveContext = func(_ context.Context, _, host string) ([]string, error) {
		return []string{"2001:db8::a1", "2001:db8::a3"}, nil
	}

	// when
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := dialer.DialContext(ctx, "tcp", "dns-is-faked.golang:12345")

	// then
	assert.Nil(t, conn)
	assert.Error(t, err)

	if dialErr, ok := err.(*ErrorSwarmDial); ok {
		assert.Equal(t, "dns-is-faked.golang:12345", dialErr.OriginalAddr)
		assert.Equal(t, ErrAllDialsFailed, dialErr.Cause)
		assert.Len(t, dialErr.DialErrors, 3)
	} else {
		assert.Fail(t, "expected to fail with ErrorSwarmDial")
	}
}

func Test_DialerSwarm_CustomResolverIsCancelable(t *testing.T) {
	dialer := NewDialerSwarm("127.0.0.1")
	dialer.ResolveContext = func(_ context.Context, _, host string) ([]string, error) {
		return []string{}, nil
	}

	// when
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		time.Sleep(1 * time.Millisecond)
		cancel()
		wg.Done()
	}()

	conn, err := dialer.DialContext(ctx, "tcp", "dns-is-faked.golang:12345")

	// then
	assert.Nil(t, conn)
	assert.Error(t, err)

	if dialErr, ok := err.(*ErrorSwarmDial); ok {
		assert.Equal(t, "dns-is-faked.golang:12345", dialErr.OriginalAddr)
		assert.Equal(t, context.Canceled, dialErr.Cause)
		assert.Len(t, dialErr.DialErrors, 0)
	} else {
		assert.Fail(t, "expected to fail with ErrorSwarmDial")
	}

	wg.Wait()
}
