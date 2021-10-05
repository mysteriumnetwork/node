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

	"github.com/mysteriumnetwork/node/requests/resolver"
)

func Test_DialerSwarm_UsesDefaultResolver(t *testing.T) {
	// given
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// when
	dialer := NewDialerSwarm("127.0.0.1", 0)
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

	dialer := NewDialerSwarm("127.0.0.1", 0)
	dialer.ResolveContext = resolver.NewResolverMap(map[string][]string{
		"dns-is-faked.golang": {"127.0.0.1", "2001:db8::a3"},
	})

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

	dialer := NewDialerSwarm("127.0.0.1", 0)
	dialer.ResolveContext = resolver.NewResolverMap(map[string][]string{
		"dns-is-faked.golang": {"2001:db8::a3", "127.0.0.1"},
	})

	// when
	conn, err := dialer.DialContext(context.Background(), "tcp", "dns-is-faked.golang:12345")

	// then
	assert.NotNil(t, conn)
	assert.NoError(t, err)
}

func Test_DialerSwarm_CustomResolverWithAllUnreachableIPs(t *testing.T) {
	dialer := NewDialerSwarm("127.0.0.1", 0)
	dialer.ResolveContext = resolver.NewResolverMap(map[string][]string{
		"dns-is-faked.golang": {"2001:db8::a1", "2001:db8::a3"},
	})

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
		assert.Failf(t, "expected to fail with ErrorSwarmDial", "but got: %v", err)
	}
}

func Test_DialerSwarm_CustomDialingIsCancelable(t *testing.T) {
	// configure lagging dialer
	dialer := NewDialerSwarm("127.0.0.1", 0)
	dialer.ResolveContext = resolver.NewResolverMap(map[string][]string{})
	dialer.Dialer = func(ctx context.Context, _, _ string) (net.Conn, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
			return nil, nil
		}
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
	} else {
		assert.Failf(t, "expected to fail with ErrorSwarmDial", "but got: %v", err)
	}

	wg.Wait()
}

func Test_DialerSwarm_CustomResolvingIsCancelable(t *testing.T) {
	// configure lagging dialer
	dialer := NewDialerSwarm("127.0.0.1", 0)
	dialer.ResolveContext = func(ctx context.Context, _, _ string) ([]string, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
			return nil, nil
		}
	}

	// when
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		time.Sleep(4 * time.Millisecond)
		cancel()
		wg.Done()
	}()

	conn, err := dialer.DialContext(ctx, "tcp", "dns-is-faked.golang:12345")

	// then
	assert.Nil(t, conn)
	assert.Equal(t, &net.OpError{Op: "dial", Net: "tcp", Source: nil, Addr: nil, Err: context.Canceled}, err)

	wg.Wait()
}

func Test_isIP(t *testing.T) {
	type args struct {
		addr string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "detects ipv4 correctly",
			args: args{
				addr: "95.216.204.232:443",
			},
			want: true,
		},
		{
			name: "detects ipv6 correctly",
			args: args{
				addr: "[2001:db8::1]:8080",
			},
			want: true,
		},
		{
			name: "detects ipv4 with no port correctly",
			args: args{
				addr: "95.216.204.232",
			},
			want: true,
		},
		{
			name: "detects ipv6 with no port correctly",
			args: args{
				addr: "::1",
			},
			want: true,
		},
		{
			name: "detects url correctly",
			args: args{
				addr: "testnet3-location.mysterium.network:443",
			},
			want: false,
		},
		{
			name: "detects url with no port correctly",
			args: args{
				addr: "testnet3-location.mysterium.network",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIP(tt.args.addr); got != tt.want {
				t.Errorf("isIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
