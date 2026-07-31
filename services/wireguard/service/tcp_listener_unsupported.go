//go:build !linux

package service

import (
	"net"
	"strconv"
)

func listenServiceTCP(bindIP net.IP, _ string, port int) (net.Listener, error) {
	return net.Listen("tcp", net.JoinHostPort(bindIP.String(), strconv.Itoa(port)))
}
