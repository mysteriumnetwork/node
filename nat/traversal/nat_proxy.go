/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package traversal

import (
	"fmt"
	"io"
	"net"
	"sync"

	log "github.com/cihub/seelog"

	"github.com/mysteriumnetwork/node/services"
)

const logPrefix = "[NATProxy] "
const bufferLen = 30000

// NATProxy provides traffic proxying functionality for registered services
type NATProxy struct {
	servicePorts  map[services.ServiceType]int
	addrLast      *net.UDPAddr
	socketProtect func(socket int) bool
	once          sync.Once
}

func (np *NATProxy) consumerHandOff(consumerPort int, remoteConn *net.UDPConn) chan struct{} {
	stop := make(chan struct{})
	if np.socketProtect == nil {
		// shutdown pinger session since openvpn client will connect directly (without NATProxy)
		remoteConn.Close()
		return stop
	}
	go np.consumerProxy(consumerPort, remoteConn, stop)
	return stop
}

// consumerProxy launches listener on pinger port and wait for openvpn connect
// Read from listener socket and write to remoteConn
// Read from remoteConn and write to listener socket
func (np *NATProxy) consumerProxy(consumerPort int, remoteConn *net.UDPConn, stop chan struct{}) {
	log.Info(logPrefix, "Inside consumer NATProxy")

	laddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", consumerPort+1))
	if err != nil {
		log.Error(logPrefix, "failed to get local address for consumer NATProxy: ", err)
		return
	}

	remoteConn.SetReadBuffer(bufferLen)
	remoteConn.SetWriteBuffer(bufferLen)

	fd, err := remoteConn.File()
	if err != nil {
		log.Error(logPrefix, "failed to fetch fd from: ", remoteConn)
		return
	}
	defer fd.Close()

	log.Info(logPrefix, "protecting socket: ", int(fd.Fd()))

	np.socketProtect(int(fd.Fd()))

	for {
		log.Info(logPrefix, "waiting connect from openvpn3 client process")
		// If for some reason consumer disconnects, new connection will be from different port
		proxyConn, err := net.ListenUDP("udp4", laddr)
		if err != nil {
			log.Errorf("%sfailed to listen for consumer proxy on: %v, %v", logPrefix, laddr, err)
			return
		}

		select {
		case <-stop:
			log.Info(logPrefix, "Stopping NATProxy handOff loop")
			proxyConn.Close()
			remoteConn.Close()
			return
		default:
			proxyConn.SetReadBuffer(bufferLen)
			proxyConn.SetWriteBuffer(bufferLen)

			np.joinUDPStreams(proxyConn, remoteConn, stop)

			proxyConn.Close()
		}
	}
}

func (np *NATProxy) joinUDPStreams(conn *net.UDPConn, remoteConn *net.UDPConn, stop chan struct{}) {
	log.Info(logPrefix, "start copying stream from consumer NATProxy to remote remoteConn")
	for {
		select {
		case <-stop:
			log.Info(logPrefix, "Stopping NATProxy joinUDPStreams")
			return
		default:
		}
		var buf [bufferLen]byte
		n, addr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			log.Errorf("%sFailed to read local process: %s cause: %s", logPrefix, conn.LocalAddr().String(), err)
			return
		}
		if n > 0 {
			_, err := remoteConn.Write(buf[:n])
			if err != nil {
				log.Errorf("%sFailed to write remote peer: %s cause: %s", logPrefix, remoteConn.RemoteAddr().String(), err)
				return
			}
			if np.addrLast != addr {
				np.addrLast = addr
				go np.readWriteToAddr(remoteConn, conn, addr, stop)
			}
		}
	}
}

func (np *NATProxy) readWriteToAddr(conn *net.UDPConn, remoteConn *net.UDPConn, addr *net.UDPAddr, stop chan struct{}) {
	for {
		select {
		case <-stop:
			log.Info(logPrefix, "Stopping NATProxy readWriteToAddr loop")
			return
		default:
		}

		var buf [bufferLen]byte
		n, err := conn.Read(buf[0:])
		if err != nil {
			log.Errorf("%sFailed to read remote peer: %s cause: %s", logPrefix, conn.LocalAddr().String(), err)
			return
		}
		if n > 0 {
			_, err := remoteConn.WriteToUDP(buf[:n], addr)
			if err != nil {
				log.Errorf("%sFailed to write to local process: %s cause: %s", logPrefix, remoteConn.LocalAddr().String(), err)
				return
			}
		}
	}
}

// NewNATProxy constructs an instance of NATProxy
func NewNATProxy() *NATProxy {
	return &NATProxy{
		servicePorts: make(map[services.ServiceType]int),
	}
}

// handOff traffic incoming through NATPinger punched hole should be handed off to NATPoxy
func (np *NATProxy) handOff(serviceType services.ServiceType, incomingConn *net.UDPConn) {
	proxyConn, err := np.getConnection(serviceType)
	if err != nil {
		log.Error(logPrefix, "failed to connect to NATProxy: ", err)
		return
	}
	log.Info(logPrefix, "handing off a connection to a service on ", proxyConn.RemoteAddr().String())
	go copyStreams(proxyConn, incomingConn)
	go copyStreams(incomingConn, proxyConn)
}

func copyStreams(dstConn *net.UDPConn, srcConn *net.UDPConn) {
	defer dstConn.Close()
	defer srcConn.Close()
	totalBytes, err := io.Copy(dstConn, srcConn)
	if err != nil {
		log.Error(logPrefix, "failed to writing / reading a stream to/from NATProxy: ", err)
	}
	log.Tracef("%stotal bytes transferred from %s to %s: %d", logPrefix,
		srcConn.RemoteAddr().String(),
		dstConn.RemoteAddr().String(),
		totalBytes)
}

func (np *NATProxy) registerServicePort(serviceType services.ServiceType, port int) {
	log.Infof("%sregistering service %s for port %d to NATProxy", logPrefix, serviceType, port)
	np.servicePorts[serviceType] = port
}

func (np *NATProxy) getConnection(serviceType services.ServiceType) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", np.servicePorts[serviceType]))
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", nil, udpAddr)
}

func (np *NATProxy) isAvailable(serviceType services.ServiceType) bool {
	return np.servicePorts[serviceType] > 0
}

func (np *NATProxy) setProtectSocketCallback(socketProtect func(socket int) bool) {
	np.socketProtect = socketProtect
}
