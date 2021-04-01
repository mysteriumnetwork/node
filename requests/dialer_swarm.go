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
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/router"
)

// ErrAllDialsFailed is returned when connecting to a peer has ultimately failed.
var ErrAllDialsFailed = errors.New("all dials failed")

// DialerSwarm is a dials to multiple addresses in parallel and earliest successful connection wins.
type DialerSwarm struct {
	// ResolveContext specifies the resolve function for doing custom DNS lookup.
	// If ResolveContext is nil, then the transport dials using package net.
	ResolveContext ResolveContext

	// Dialer specifies the dial function for creating unencrypted TCP connections.
	Dialer DialContext

	// dnsHeadstart specifies the time delay that requests via IP incur.
	dnsHeadstart time.Duration
}

// NewDialerSwarm creates swarm dialer with default configuration.
func NewDialerSwarm(srcIP string, dnsHeadstart time.Duration) *DialerSwarm {
	return &DialerSwarm{
		dnsHeadstart: dnsHeadstart,
		Dialer: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
			LocalAddr: &net.TCPAddr{IP: net.ParseIP(srcIP)},
			Control: func(net, address string, c syscall.RawConn) (err error) {
				err = c.Control(func(f uintptr) {
					fd := int(f)
					err := router.Protect(fd)
					if err != nil {
						log.Error().Err(err).Msg("Failed to protect connection")
					}
				})
				return err
			},
		}).DialContext,
	}
}

// DialContext connects to the address on the named network using the provided context.
func (ds *DialerSwarm) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if ds.ResolveContext != nil {
		addrs, err := ds.ResolveContext(ctx, network, addr)
		if err != nil {
			return nil, &net.OpError{Op: "dial", Net: network, Source: nil, Addr: nil, Err: err}
		}

		conn, errDial := ds.dialAddrs(ctx, network, addrs)
		if errDial != nil {
			errDial.OriginalAddr = addr

			return nil, errDial
		}

		return conn, nil
	}

	return ds.Dialer(ctx, network, addr)
}

func (ds *DialerSwarm) dialAddrs(ctx context.Context, network string, addrs []string) (net.Conn, *ErrorSwarmDial) {
	addrChan := make(chan string, len(addrs))
	for _, addr := range addrs {
		addrChan <- addr
	}

	close(addrChan)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultCh := make(chan dialResult)
	err := &ErrorSwarmDial{}

	var active int
dialLoop:
	for addrChan != nil || active > 0 {
		// Check for context cancellations and/or responses first.
		select {
		// Overall dialing canceled.
		case <-ctx.Done():
			break dialLoop

		// Some dial result arrived.
		case resp := <-resultCh:
			active--
			if resp.Err != nil {
				err.addErr(resp.Addr, resp.Err)
			} else if resp.Conn != nil {
				return resp.Conn, nil
			}

			continue

		default:
		}

		// Now, attempt to dial.
		select {
		case addr, ok := <-addrChan:
			if !ok {
				addrChan = nil

				continue
			}

			// Prefer dialing via dns, give them a head start.
			if !isIP(addr) {
				go ds.dialAddr(ctx, network, addr, resultCh)
			} else {
				go func() {
					select {
					case <-time.After(ds.dnsHeadstart):
						break
					case <-ctx.Done():
						return
					}
					ds.dialAddr(ctx, network, addr, resultCh)
				}()
			}

			active++

		case <-ctx.Done():
			break dialLoop

		case resp := <-resultCh:
			active--
			if resp.Err != nil {
				err.addErr(resp.Addr, resp.Err)
			} else if resp.Conn != nil {
				return resp.Conn, nil
			}
		}
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		err.Cause = ctxErr
	} else {
		err.Cause = ErrAllDialsFailed
	}

	return nil, err
}

func isIP(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		ip := net.ParseIP(addr)
		return ip != nil
	}
	ip := net.ParseIP(host)
	return ip != nil
}

func (ds *DialerSwarm) dialAddr(ctx context.Context, network, addr string, resp chan dialResult) {
	// Dialing might be canceled already.
	if ctx.Err() != nil {
		return
	}

	conn, err := ds.Dialer(ctx, network, addr)
	select {
	case resp <- dialResult{Conn: conn, Addr: addr, Err: err}:
	case <-ctx.Done():
		if err == nil {
			conn.Close()
		}
	}
}

type dialResult struct {
	Conn net.Conn
	Addr string
	Err  error
}

// ErrorSwarmDial is the error type returned when dialing multiple addresses.
type ErrorSwarmDial struct {
	OriginalAddr string
	DialErrors   []ErrorDial
	Cause        error
}

func (e *ErrorSwarmDial) addErr(addr string, err error) {
	e.DialErrors = append(e.DialErrors, ErrorDial{
		Addr:  addr,
		Cause: err,
	})
}

// Error returns string equivalent for error.
func (e *ErrorSwarmDial) Error() string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "failed to dial %s:", e.OriginalAddr)

	if e.Cause != nil {
		fmt.Fprintf(&builder, " %s", e.Cause)
	}

	for _, te := range e.DialErrors {
		fmt.Fprintf(&builder, "\n  * [%s] %s", te.Addr, te.Cause)
	}

	return builder.String()
}

// Unwrap unwraps the original err for use with errors.Unwrap.
func (e *ErrorSwarmDial) Unwrap() error {
	return e.Cause
}

// ErrorDial is the error returned when dialing a specific address.
type ErrorDial struct {
	Addr  string
	Cause error
}

// Error returns string equivalent for error.
func (e *ErrorDial) Error() string {
	return fmt.Sprintf("failed to dial %s: %s", e.Addr, e.Cause)
}

// Unwrap unwraps the original err for use with errors.Unwrap.
func (e *ErrorDial) Unwrap() error {
	return e.Cause
}
