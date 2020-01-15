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
	"time"

	"github.com/rs/zerolog/log"
)

const bufferLen = 2048 * 1024

// NATProxy provides traffic proxying functionality for registered services
type NATProxy struct {
	servicePorts  map[string]int
	addrLast      *net.UDPAddr
	socketProtect func(socket int) bool
}

func (np *NATProxy) consumerHandOff(consumerAddr string, remoteConn *net.UDPConn) chan struct{} {
	time.Sleep(400 * time.Millisecond)
	stop := make(chan struct{})
	if np.socketProtect == nil {
		// shutdown pinger session since openvpn client will connect directly (without NATProxy)
		remoteConn.Close()
		return stop
	}
	go np.consumerProxy(consumerAddr, remoteConn, stop)
	return stop
}

// consumerProxy launches listener on pinger port and wait for openvpn connect
// Read from listener socket and write to remoteConn
// Read from remoteConn and write to listener socket
func (np *NATProxy) consumerProxy(consumerAddr string, remoteConn *net.UDPConn, stop chan struct{}) {
	log.Info().Msg("Inside consumer NATProxy")

	laddr, err := net.ResolveUDPAddr("udp4", consumerAddr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get local address for consumer NATProxy")
		return
	}

	remoteConn.SetReadBuffer(bufferLen)
	remoteConn.SetWriteBuffer(bufferLen)

	fd, err := remoteConn.File()
	if err != nil {
		log.Error().Msgf("Failed to fetch fd from: %v", remoteConn)
		return
	}
	defer fd.Close()

	log.Info().Msgf("Protecting socket: %d", int(fd.Fd()))

	np.socketProtect(int(fd.Fd()))

	for {
		log.Info().Msg("Waiting connect from openvpn3 client process")
		// If for some reason consumer disconnects, new connection will be from different port
		proxyConn, err := net.ListenUDP("udp4", laddr)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to listen for consumer proxy on: %v", laddr)
			return
		}

		select {
		case <-stop:
			log.Info().Msg("Stopping NATProxy handOff loop")
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
	log.Info().Msg("Start copying stream from consumer NATProxy to remote remoteConn")
	buf := make([]byte, bufferLen)
	for {
		select {
		case <-stop:
			log.Info().Msg("Stopping NATProxy joinUDPStreams")
			return
		default:
		}
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Error().Err(err).Msg("Failed to read local process: " + conn.LocalAddr().String())
			return
		}
		if n > 0 {
			_, err := remoteConn.Write(buf[:n])
			if err != nil {
				log.Error().Err(err).Msg("Failed to write remote peer: " + remoteConn.RemoteAddr().String())
				return
			}
			if np.addrLast.String() != addr.String() {
				np.addrLast = addr
				go np.readWriteToAddr(remoteConn, conn, addr, stop)
			}
		}
	}
}

func (np *NATProxy) readWriteToAddr(conn *net.UDPConn, remoteConn *net.UDPConn, addr *net.UDPAddr, stop chan struct{}) {
	buf := make([]byte, bufferLen)
	for {
		select {
		case <-stop:
			log.Info().Msg("Stopping NATProxy readWriteToAddr loop")
			return
		default:
		}

		n, err := conn.Read(buf)
		if err != nil {
			log.Error().Err(err).Msg("Failed to read remote peer: " + conn.LocalAddr().String())
			return
		}
		for i := 0; i < n; {
			written, err := remoteConn.WriteToUDP(buf[i:n], addr)
			if written < n {
				log.Debug().Msgf("Partial write of %d bytes", written)
			}
			if err != nil {
				log.Error().Err(err).Msg("Failed to write to local process: " + remoteConn.LocalAddr().String())
				return
			}
			i += written
		}
	}
}

// NewNATProxy constructs an instance of NATProxy
func NewNATProxy() *NATProxy {
	return &NATProxy{
		servicePorts: make(map[string]int),
	}
}

// handOff traffic incoming through NATPinger punched hole should be handed off to NATPoxy
func (np *NATProxy) handOff(key string, incomingConn *net.UDPConn) {
	proxyConn, err := np.getConnection(key)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to NATProxy")
		return
	}
	log.Info().Msg("Handing off a connection to a service on " + proxyConn.RemoteAddr().String())
	go copyStreams(proxyConn, incomingConn)
	go copyStreams(incomingConn, proxyConn)
}

func copyStreams(dstConn *net.UDPConn, srcConn *net.UDPConn) {
	buf := make([]byte, bufferLen)

	defer dstConn.Close()
	defer srcConn.Close()
	totalBytes, err := io.CopyBuffer(dstConn, srcConn, buf)
	if err != nil {
		log.Error().Err(err).Msg("Failed to write/read a stream to/from NATProxy")
	}
	log.Debug().Msgf("Total bytes transferred from %s to %s: %d",
		srcConn.RemoteAddr().String(),
		dstConn.RemoteAddr().String(),
		totalBytes)
}

func (np *NATProxy) registerServicePort(key string, port int) {
	log.Info().Msgf("Registering service %s for port %d to NATProxy", key, port)
	np.servicePorts[key] = port
}

func (np *NATProxy) getConnection(key string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", np.servicePorts[key]))
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", nil, udpAddr)
}

func (np *NATProxy) isAvailable(key string) bool {
	return np.servicePorts[key] > 0
}

func (np *NATProxy) setProtectSocketCallback(socketProtect func(socket int) bool) {
	np.socketProtect = socketProtect
}
