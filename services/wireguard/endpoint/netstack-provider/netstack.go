/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package netstack_provider

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/config"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
	"golang.zx2c4.com/wireguard/tun"
	"gvisor.dev/gvisor/pkg/bufferv2"
	"gvisor.dev/gvisor/pkg/refs"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

type netTun struct {
	stack          *stack.Stack
	dispatcher     stack.NetworkDispatcher
	events         chan tun.Event
	incomingPacket chan *bufferv2.View
	mtu            int
	dnsPort        int
	localAddresses []netip.Addr

	limiter *rate.Limiter
}

type (
	endpoint netTun
	Net      netTun
)

func (e *endpoint) Attach(dispatcher stack.NetworkDispatcher) {
	e.dispatcher = dispatcher
}

func (e *endpoint) IsAttached() bool {
	return e.dispatcher != nil
}

func (e *endpoint) MTU() uint32 {
	mtu, err := (*netTun)(e).MTU()
	if err != nil {
		panic(err)
	}
	return uint32(mtu)
}

func (*endpoint) Capabilities() stack.LinkEndpointCapabilities {
	return stack.CapabilityNone
}

func (*endpoint) MaxHeaderLength() uint16 {
	return 0
}

func (*endpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

func (*endpoint) Wait() {}

func (e *endpoint) WritePacket(pkt *stack.PacketBuffer) tcpip.Error {
	e.incomingPacket <- pkt.ToView()
	return nil
}

func (e *endpoint) WritePackets(pl stack.PacketBufferList) (size int, err tcpip.Error) {
	for _, pkt := range pl.AsSlice() {
		size += pkt.Size()
		err = e.WritePacket(pkt)
		if err != nil {
			return size, err
		}
	}

	return size, nil
}

func (e *endpoint) WriteRawPacket(*stack.PacketBuffer) tcpip.Error {
	panic("not implemented")
}

func (*endpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (e *endpoint) AddHeader(p *stack.PacketBuffer) {
}

func CreateNetTUN(localAddresses []netip.Addr, dnsPort, mtu int) (tun.Device, *Net, error) {
	refs.SetLeakMode(refs.NoLeakChecking)

	opts := stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol, ipv6.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol, icmp.NewProtocol6, icmp.NewProtocol4},
	}

	var limiter *rate.Limiter
	if config.GetBool(config.FlagShaperEnabled) {
		log.Warn().Msgf("Shaper bandwidth: %v", config.GetUInt64(config.FlagShaperBandwidth))
		bandwidthBytes := config.GetUInt64(config.FlagShaperBandwidth) * 1024
		limiter = rate.NewLimiter(rate.Limit(bandwidthBytes), int(bandwidthBytes))
	}

	dev := &netTun{
		stack:          stack.New(opts),
		events:         make(chan tun.Event, 10),
		incomingPacket: make(chan *bufferv2.View),
		mtu:            mtu,
		dnsPort:        dnsPort,
		localAddresses: localAddresses,
		limiter:        limiter,
	}

	tcpFwd := tcp.NewForwarder(dev.stack, 0, 10000, dev.acceptTCP)
	dev.stack.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpFwd.HandlePacket)

	udpFwd := udp.NewForwarder(dev.stack, dev.acceptUDP)
	dev.stack.SetTransportProtocolHandler(udp.ProtocolNumber, udpFwd.HandlePacket)

	tcpipErr := dev.stack.CreateNIC(1, (*endpoint)(dev))
	if tcpipErr != nil {
		return nil, nil, fmt.Errorf("failed to create netstack NIC %v", tcpipErr)
	}

	for _, ip := range localAddresses {
		if err := dev.addAddress(tcpip.Address(ip.AsSlice())); err != nil {
			return nil, nil, fmt.Errorf("failed to add local address (%v): %v", ip, tcpipErr)
		}
	}

	dev.events <- tun.EventUp
	return dev, (*Net)(dev), nil
}

func CreateNetTUNWithStack(localAddresses []netip.Addr, dnsPort, mtu int) (tun.Device, *Net, *stack.Stack, error) {
	t, n, err := CreateNetTUN(localAddresses, dnsPort, mtu)

	stack := t.(*netTun).stack
	stack.SetPromiscuousMode(1, true)

	defaultRoute, _ := tcpip.NewSubnet(tcpip.Address([]byte{0, 0, 0, 0}), tcpip.AddressMask([]byte{0, 0, 0, 0}))
	stack.SetRouteTable([]tcpip.Route{
		{Destination: defaultRoute, NIC: 1},
	})

	return t, n, stack, err
}

func (tun *netTun) Name() (string, error) {
	return "go", nil
}

func (tun *netTun) File() *os.File {
	return nil
}

func (tun *netTun) Events() chan tun.Event {
	return tun.events
}

func (tun *netTun) Read(buf []byte, offset int) (int, error) {
	view, ok := <-tun.incomingPacket
	if !ok {
		return 0, os.ErrClosed
	}

	return view.Read(buf[offset:])
}

func (tun *netTun) Write(buf []byte, offset int) (int, error) {
	packet := buf[offset:]
	if len(packet) == 0 {
		return 0, nil
	}

	pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Payload: bufferv2.MakeWithData(append([]byte(nil), packet...)),
	})

	switch packet[0] >> 4 {
	case 4:
		tun.dispatcher.DeliverNetworkPacket(ipv4.ProtocolNumber, pkb)
	case 6:
		tun.dispatcher.DeliverNetworkPacket(ipv6.ProtocolNumber, pkb)
	}

	return len(buf), nil
}

func (tun *netTun) Flush() error {
	return nil
}

func (tun *netTun) Close() error {
	tun.stack.RemoveNIC(1)
	tun.stack.Close()

	if tun.events != nil {
		close(tun.events)
	}

	if tun.incomingPacket != nil {
		close(tun.incomingPacket)
	}

	return nil
}

func (tun *netTun) MTU() (int, error) {
	return tun.mtu, nil
}

func (tun *netTun) addAddress(ip tcpip.Address) error {
	protoAddr := tcpip.ProtocolAddress{
		Protocol:          ipv4.ProtocolNumber,
		AddressWithPrefix: ip.WithPrefix(),
	}
	tcpipErr := tun.stack.AddProtocolAddress(1, protoAddr, stack.AddressProperties{
		PEB:        stack.CanBePrimaryEndpoint,
		ConfigType: stack.AddressConfigStatic,
	})
	if tcpipErr != nil {
		return fmt.Errorf("failed to add protocol address (%v): %v", ip, tcpipErr)
	}

	return nil
}

func (tun *netTun) acceptTCP(r *tcp.ForwarderRequest) {
	if isPrivateIP(net.IP(r.ID().LocalAddress)) {
		log.Warn().Msgf("Access to private IPv4 subnet is restricted: %s", r.ID().LocalAddress.String())
		return
	}

	reqDetails := r.ID()

	tun.addAddress(reqDetails.LocalAddress)

	var wq waiter.Queue
	ep, tcpErr := r.CreateEndpoint(&wq)
	if tcpErr != nil {
		log.Error().Err(fmt.Errorf(tcpErr.String())).Msg("Failed to create TCP endpoint for forwarding request")
		r.Complete(true)
		return
	}
	r.Complete(false)

	ep.SocketOptions().SetKeepAlive(true)

	client := gonet.NewTCPConn(&wq, ep)
	defer client.Close()

	dialAddrStr := fmt.Sprintf("%s:%d", reqDetails.LocalAddress, reqDetails.LocalPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stdDialer net.Dialer

	server, err := stdDialer.DialContext(ctx, "tcp", dialAddrStr)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to local server at %s", dialAddrStr)
		return
	}
	defer server.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go tun.relay(&wg, server, client)
	go tun.relay(&wg, client, server)
	wg.Wait()
}

func (tun *netTun) relay(wg *sync.WaitGroup, dst, src net.Conn) {
	defer wg.Done()

	r := NewReader(src, tun.limiter)
	_, err := io.Copy(dst, r)
	if err != nil {
		log.Trace().Err(err).Msg("relay: data copy")
	}

	err = dst.SetReadDeadline(time.Now().Add(-1)) // make another Copy exit
	if err != nil {
		log.Trace().Err(err).Msg("relay: setting read deadline")
	}
}

func (tun *netTun) acceptUDP(req *udp.ForwarderRequest) {
	if isPrivateIP(net.IP(req.ID().LocalAddress)) {
		log.Warn().Msgf("Access to private IPv4 subnet is restricted: %s", req.ID().LocalAddress.String())
		return
	}

	sess := req.ID()

	tun.addAddress(sess.LocalAddress)

	var wq waiter.Queue

	ep, udpErr := req.CreateEndpoint(&wq)
	if udpErr != nil {
		log.Error().Err(fmt.Errorf(udpErr.String())).Msg("Failed to create UDP endpoint for forwarding request")
		return
	}

	client := gonet.NewUDPConn(tun.stack, &wq, ep)

	clientAddr := &net.UDPAddr{IP: net.IP([]byte(sess.RemoteAddress)), Port: int(sess.RemotePort)}
	remoteAddr := &net.UDPAddr{IP: net.IP([]byte(sess.LocalAddress)), Port: int(sess.LocalPort)}
	proxyAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: int(sess.RemotePort)}

	if remoteAddr.Port == 53 && tun.dnsPort > 0 && tun.isLocal(sess.LocalAddress) {
		remoteAddr.Port = tun.dnsPort
		remoteAddr.IP = net.ParseIP("127.0.0.1")
	}

	proxyConn, err := net.ListenUDP("udp", proxyAddr)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to bind local port %d, trying one more time with random port", proxyAddr)
		proxyAddr.Port = 0

		proxyConn, err = net.ListenUDP("udp", proxyAddr)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to bind local random port %s", proxyAddr)
			return
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	go tun.proxy(ctx, cancel, client, clientAddr, proxyConn) // loc <- remote
	go tun.proxy(ctx, cancel, proxyConn, remoteAddr, client) // remote <- loc
}

func (tun *netTun) isLocal(remoteAddr tcpip.Address) bool {
	for _, ip := range tun.localAddresses {
		if tcpip.Address(ip.AsSlice()) == remoteAddr {
			return true
		}
	}

	return false
}

const (
	idleTimeout = 2 * time.Minute
)

func (tun *netTun) proxy(ctx context.Context, cancel context.CancelFunc, dst net.PacketConn, dstAddr net.Addr, src net.PacketConn) {
	defer cancel()

	buf := make([]byte, tun.mtu)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			src.SetReadDeadline(time.Now().Add(idleTimeout))

			n, srcAddr, err := src.ReadFrom(buf)
			if err != nil {
				if ctx.Err() == nil {
					log.Trace().Msgf("Failed to read packed from %s", srcAddr)
				}
				return
			}

			// delay according to bandwidth limit
			if n > 0 && tun.limiter != nil {
				err := tun.limiter.WaitN(ctx, n)
				if err != nil {
					log.Trace().Msgf("Shaper error: %v", err)
					return
				}
			}

			_, err = dst.WriteTo(buf[:n], dstAddr)
			if err != nil {
				if ctx.Err() == nil {
					log.Trace().Err(err).Msgf("Failed to write packed to %s", dstAddr)
				}
				return
			}
			dst.SetReadDeadline(time.Now().Add(idleTimeout))
		}
	}
}
