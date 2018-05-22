package management

import (
	"net"
	"fmt"
	"strings"
	"strconv"
	"github.com/kataras/iris/core/errors"
)

/**
 * Used for OpenVPN management connection
 */
type Address struct {
	IP   string
	Port int
}

func (ma *Address) String() string {
	return fmt.Sprintf("%s:%d", ma.IP, ma.Port)
}

/*
 * Finds an available TCP port on 127.0.0.1
 *
 * OpenVPN management connection needs to listen on a known port
 * using this method we can figure out which port/address to use
 */
func GetAvailableAddress() (*Address, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	defer l.Close()

	// get the actual bound address/port
	address := l.Addr().(*net.TCPAddr)

	return &Address{address.IP.String(), address.Port}, nil
}

func GetPortAndAddressFromString(address string) (*Address, error) {
	str := strings.Split(address, ":")

	if len(str) < 2 {
		return nil, errors.New("Failed to parse address string.")
	}

	port, err := strconv.Atoi(str[1])

	if err != nil {
		return nil, errors.New("Failed to parse port number.")
	}

	return &Address{str[0], port}, nil
}
