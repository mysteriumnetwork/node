package netstack

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/tun"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/buffer"
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
	incomingPacket chan buffer.VectorisedView
	mtu            int
	dnsPort        int
	localAddresses []netip.Addr
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

func (e *endpoint) WritePacket(_ stack.RouteInfo, _ tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) tcpip.Error {
	e.incomingPacket <- buffer.NewVectorisedView(pkt.Size(), pkt.Views())
	return nil
}

func (e *endpoint) WritePackets(stack.RouteInfo, stack.PacketBufferList, tcpip.NetworkProtocolNumber) (int, tcpip.Error) {
	panic("not implemented")
}

func (e *endpoint) WriteRawPacket(*stack.PacketBuffer) tcpip.Error {
	panic("not implemented")
}

func (*endpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (e *endpoint) AddHeader(tcpip.LinkAddress, tcpip.LinkAddress, tcpip.NetworkProtocolNumber, *stack.PacketBuffer) {
}

func CreateNetTUN(localAddresses []netip.Addr, dnsPort, mtu int) (tun.Device, *Net, error) {
	opts := stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol, ipv6.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol, icmp.NewProtocol6, icmp.NewProtocol4},
	}
	dev := &netTun{
		stack:          stack.New(opts),
		events:         make(chan tun.Event, 10),
		incomingPacket: make(chan buffer.VectorisedView),
		mtu:            mtu,
		dnsPort:        dnsPort,
		localAddresses: localAddresses,
	}

	tcpFwd := tcp.NewForwarder(dev.stack, 0, 128, dev.acceptTCP)
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

	pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{Data: buffer.NewVectorisedView(len(packet), []buffer.View{buffer.NewViewFromBytes(packet)})})
	switch packet[0] >> 4 {
	case 4:
		tun.dispatcher.DeliverNetworkPacket("", "", ipv4.ProtocolNumber, pkb)
	case 6:
		tun.dispatcher.DeliverNetworkPacket("", "", ipv6.ProtocolNumber, pkb)
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

	connClosed := make(chan error, 2)
	go func() {
		_, err := io.Copy(server, client)
		connClosed <- err
	}()
	go func() {
		_, err := io.Copy(client, server)
		connClosed <- err
	}()

	err = <-connClosed
	if err != nil {
		log.Trace().Err(err).Msgf("Proxy connection closed: %s", dialAddrStr)
	}
}

func (tun *netTun) acceptUDP(req *udp.ForwarderRequest) {
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

	idleTimeout := 2 * time.Minute
	timeout := time.AfterFunc(idleTimeout, func() {
		log.Trace().Msgf("Session timed out, %s<->%s", proxyAddr, remoteAddr)

		cancel()
		client.Close()
		proxyConn.Close()
	})
	extend := func() {
		timeout.Reset(idleTimeout)
	}

	go tun.proxy(ctx, cancel, client, clientAddr, proxyConn, extend)
	go tun.proxy(ctx, cancel, proxyConn, remoteAddr, client, extend)
}

func (tun *netTun) isLocal(remoteAddr tcpip.Address) bool {
	for _, ip := range tun.localAddresses {
		if tcpip.Address(ip.AsSlice()) == remoteAddr {
			return true
		}
	}

	return false
}

func (tun *netTun) proxy(ctx context.Context, cancel context.CancelFunc, dst net.PacketConn, dstAddr net.Addr, src net.PacketConn, extend func()) {
	defer cancel()

	buf := make([]byte, tun.mtu)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, srcAddr, err := src.ReadFrom(buf)
			if err != nil {
				if ctx.Err() == nil {
					log.Trace().Msgf("Failed to read packed from %s", srcAddr)
				}
				return
			}

			_, err = dst.WriteTo(buf[:n], dstAddr)
			if err != nil {
				if ctx.Err() == nil {
					log.Trace().Err(err).Msgf("Failed to write packed to %s", dstAddr)
				}
				return
			}
			extend()
		}
	}
}
