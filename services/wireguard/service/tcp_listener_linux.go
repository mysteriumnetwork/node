//go:build linux

package service

import (
	"context"
	"net"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

func listenServiceTCP(bindIP net.IP, iface string, port int) (net.Listener, error) {
	listenConfig := net.ListenConfig{
		Control: func(_, _ string, raw syscall.RawConn) error {
			var optionErr error
			if err := raw.Control(func(fd uintptr) {
				optionErr = unix.SetsockoptString(int(fd), unix.SOL_SOCKET, unix.SO_BINDTODEVICE, iface)
			}); err != nil {
				return err
			}
			return optionErr
		},
	}
	return listenConfig.Listen(context.Background(), "tcp", net.JoinHostPort(bindIP.String(), strconv.Itoa(port)))
}
