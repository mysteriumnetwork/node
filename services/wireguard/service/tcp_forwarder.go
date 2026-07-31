package service

import (
	"errors"
	"io"
	"net"
	"sync"
)

type tcpForwarder struct {
	listener net.Listener
	dialer   func(port int) (net.Conn, error)
	port     int
	mu       sync.Mutex
	conns    map[net.Conn]struct{}
	closed   bool
	wg       sync.WaitGroup
}

func newTCPForwarder(bindIP net.IP, iface string, port int, dialer func(port int) (net.Conn, error)) (*tcpForwarder, error) {
	if bindIP == nil || bindIP.IsUnspecified() {
		return nil, errors.New("WireGuard gateway address is required")
	}
	if iface == "" {
		return nil, errors.New("WireGuard interface is required")
	}
	if port < 1 || port > 65535 {
		return nil, errors.New("runtime TCP port must be between 1 and 65535")
	}
	if dialer == nil {
		return nil, errors.New("isolated workload dialer is required")
	}

	listener, err := listenServiceTCP(bindIP, iface, port)
	if err != nil {
		return nil, err
	}
	forwarder := &tcpForwarder{
		listener: listener,
		dialer:   dialer,
		port:     port,
		conns:    make(map[net.Conn]struct{}),
	}
	forwarder.wg.Add(1)
	go forwarder.serve()
	return forwarder, nil
}

func (forwarder *tcpForwarder) serve() {
	defer forwarder.wg.Done()
	for {
		client, err := forwarder.listener.Accept()
		if err != nil {
			return
		}
		forwarder.mu.Lock()
		if forwarder.closed {
			forwarder.mu.Unlock()
			_ = client.Close()
			return
		}
		forwarder.conns[client] = struct{}{}
		forwarder.mu.Unlock()
		forwarder.wg.Add(1)
		go func() {
			defer forwarder.wg.Done()
			defer func() {
				_ = client.Close()
				forwarder.mu.Lock()
				delete(forwarder.conns, client)
				forwarder.mu.Unlock()
			}()
			server, err := forwarder.dialer(forwarder.port)
			if err != nil {
				return
			}
			forwarder.mu.Lock()
			if forwarder.closed {
				forwarder.mu.Unlock()
				_ = server.Close()
				return
			}
			forwarder.conns[server] = struct{}{}
			forwarder.mu.Unlock()
			defer func() {
				_ = server.Close()
				forwarder.mu.Lock()
				delete(forwarder.conns, server)
				forwarder.mu.Unlock()
			}()
			relayTCP(client, server)
		}()
	}
}

func relayTCP(left, right net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	copyConn := func(dst, src net.Conn) {
		defer wg.Done()
		_, _ = io.Copy(dst, src)
		if tcp, ok := dst.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
	}
	go copyConn(left, right)
	go copyConn(right, left)
	wg.Wait()
}

func (forwarder *tcpForwarder) Close() error {
	_ = forwarder.listener.Close()
	forwarder.mu.Lock()
	forwarder.closed = true
	for conn := range forwarder.conns {
		_ = conn.Close()
	}
	forwarder.mu.Unlock()
	forwarder.wg.Wait()
	return nil
}
