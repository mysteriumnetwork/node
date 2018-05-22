package management

import (
	"net"
	"fmt"
	"strings"
	"strconv"
)

type Address struct {
	IP   string
	Port int
}

func (ma *Address) String() string {
	return fmt.Sprintf("%s:%d", ma.IP, ma.Port)
}

func GetAvailableAddress() (*Address, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	defer l.Close()

	address := l.Addr().(*net.TCPAddr)

	return &Address{address.IP.String(), address.Port}, nil
}

func GetAddressFromString(address string) Address {
	str := strings.Split(address, ":")
	port, _ := strconv.Atoi(str[1])

	return Address{
		IP:   str[0],
		Port: port,
	}
}
