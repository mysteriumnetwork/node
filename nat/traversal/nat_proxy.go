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

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/services"
)

const logPrefix = "[NATProxy] "

// NATProxy provides traffic proxying functionality for registered services
type NATProxy struct {
	servicePorts map[services.ServiceType]int
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
	log.Infof("%sregistering service %s for port %d to NAT proxy", prefix, serviceType, port)
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
	if np.servicePorts[serviceType] > 0 {
		return true
	}
	return false
}
