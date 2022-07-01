package handler

import (
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	ipv4_ "golang.org/x/net/ipv4"
	"golang.org/x/time/rate"

	log "github.com/sirupsen/logrus"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/header/parse"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type pingEnt struct {
	timeout time.Time

	dst tcpip.Address
	src tcpip.Address
}

// Correlate request /reply by: <dest, id, seq, header.Checksum(replyData, 0)>
type requestID struct {
	dstAddr      string
	id           int
	seq          int
	dataChecksum uint16 // reply contains the same data
}

func getRequestID(msg *icmp.Echo, dst tcpip.Address) requestID {
	r := requestID{
		dstAddr:      string(dst),
		seq:          msg.Seq,
		id:           msg.ID,
		dataChecksum: header.Checksum(msg.Data, 0),
	}
	return r
}

type Pinger struct {
	c *icmp.PacketConn

	entries map[requestID]*pingEnt
	mu      sync.Mutex
	limiter *rate.Limiter

	s *stack.Stack
}

func newPinger(s *stack.Stack) *Pinger {
	pingMap := make(map[requestID]*pingEnt)

	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatalf("listen err, %s", err)
	}

	return &Pinger{
		c:       c,
		entries: pingMap,
		s:       s,
		limiter: rate.NewLimiter(1, 2),
	}
}

// send to external network
func (p *Pinger) Ping(msg *icmp.Echo, dst, src tcpip.Address) {
	wm := icmp.Message{
		Type: ipv4_.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   msg.ID,
			Seq:  msg.Seq,
			Data: msg.Data,
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := p.c.WriteTo(wb, &net.IPAddr{IP: net.IP(dst)}); err != nil {
		log.Fatalf("WriteTo err, %s", err)
	}

	rid := getRequestID(msg, dst)
	p.mu.Lock()
	p.entries[rid] = &pingEnt{
		src:     src,
		dst:     dst,
		timeout: time.Now().Add(10 * time.Second),
	}
	p.mu.Unlock()
}

func (p *Pinger) clearOldEntriesJob() {
	for {
		time.Sleep(5 * time.Second)

		tnow := time.Now()
		p.mu.Lock()
		for k, v := range p.entries {
			if tnow.After(v.timeout) {
				// log.Println("clearOldEntriesJob>", k)
				delete(p.entries, k)
			}
		}
		p.mu.Unlock()
	}
}

// process echo reply
func (p *Pinger) ProcessIncomingICMP() {
	rb := make([]byte, 1500)
	for {
		// notice: cm is not avail. on Windows
		n, cm, peer, err := p.c.IPv4PacketConn().ReadFrom(rb)
		if err != nil {
			log.Fatal(err)
		}
		_ = cm

		rm, err := icmp.ParseMessage(ipv4_.ICMPTypeEchoReply.Protocol(), rb[:n])
		if err != nil {
			log.Fatal(err)
		}

		switch rm.Type {
		case ipv4_.ICMPTypeEchoReply:
			log.Printf("got reflection from %v. %v > ", peer)

			msg := rm.Body.(*icmp.Echo)
			ip := net.ParseIP(peer.String())
			reqID := getRequestID(msg, tcpip.Address(ip.To4()))

			// log.Printf("> %v %v", peer, reqID)
			p.mu.Lock()
			ent, ok := p.entries[reqID]
			if ok {
				delete(p.entries, reqID)
			}
			p.mu.Unlock()

			if ok {
				log.Println("Reply >", ent, reqID)

				view := wrapMsgIntoIPv4Packet(ent.dst, ent.src, rb[:n])
				p.s.WritePacketToRemote(1, "", ipv4.ProtocolNumber, view)
			}

		default:
			log.Printf("got %+v; want echo reply", rm)
		}
	}
}

func ICMPHandler(s *stack.Stack) func(id stack.TransportEndpointID, pkt *stack.PacketBuffer) bool {
	pinger := newPinger(s)
	go pinger.ProcessIncomingICMP()
	go pinger.clearOldEntriesJob()

	return func(id stack.TransportEndpointID, pkt *stack.PacketBuffer) bool {
		log.Infof("[ICMP] Receive a icmp package, SRC: %s, DST: %s", id.LocalAddress, id.RemoteAddress)

		// // remote - peer's tunnnel interface address
		if id.LocalAddress.String() == "10.0.0.1" {
			log.Info("[ICMP] handle localy")
			_, view := handleICMP(pkt)
			s.WritePacketToRemote(1, "", ipv4.ProtocolNumber, view)
			return true
		}

		// forward it to remote
		icmpMsg := stack.PayloadSince(pkt.TransportHeader())
		rmsg, err := icmp.ParseMessage(ipv4_.ICMPTypeEcho.Protocol(), icmpMsg)
		if err != nil {
			log.Fatal(err)
		}
		// iph := header.IPv4(pkt.NetworkHeader().View())

		switch rmsg.Body.(type) {
		case *icmp.Echo:
			if pinger.limiter.Allow() {
				msg := rmsg.Body.(*icmp.Echo)
				log.Println("icmp echo > forward", msg.ID, msg.Seq)
				pinger.Ping(msg, id.LocalAddress, id.RemoteAddress)
			} else {
				log.Println("icmp echo > rate limit exceeded, dropping")
			}

		default:
			log.Printf("received %+v from %v; wanted echo", rmsg, id.RemoteAddress)
			return false
		}
		return true
	}
}

func wrapMsgIntoIPv4Packet(src, dst tcpip.Address, msg []byte) buffer.VectorisedView {

	view := buffer.NewView(header.IPv4MinimumSize)
	replyIPHdr := header.IPv4(view)

	replyIPHdr.Encode(&header.IPv4Fields{
		TTL:      64,
		SrcAddr:  src,
		DstAddr:  dst,
		Protocol: 1,
	})

	replyIPHdr.SetTotalLength(uint16(len(replyIPHdr) + len(msg)))
	replyIPHdr.SetChecksum(0)
	replyIPHdr.SetChecksum(^replyIPHdr.CalculateChecksum())

	replyVV := buffer.View(replyIPHdr).ToVectorisedView()
	replyVV.AppendView(buffer.View(msg))
	return replyVV
}

func handleICMP(pkt *stack.PacketBuffer) (*stack.PacketBuffer, buffer.VectorisedView) {
	replyData := stack.PayloadSince(pkt.TransportHeader())
	iph := header.IPv4(pkt.NetworkHeader().View())

	replyHeaderLength := uint8(header.IPv4MinimumSize)
	replyIPHdrBytes := make([]byte, 0, replyHeaderLength)
	replyIPHdrBytes = append(replyIPHdrBytes, iph[:header.IPv4MinimumSize]...)

	replyIPHdr := header.IPv4(replyIPHdrBytes)
	replyIPHdr.SetHeaderLength(replyHeaderLength)
	replyIPHdr.SetSourceAddress(iph.DestinationAddress())
	replyIPHdr.SetDestinationAddress(iph.SourceAddress())
	replyIPHdr.SetTTL(iph.TTL())
	replyIPHdr.SetTotalLength(uint16(len(replyIPHdr) + len(replyData)))
	replyIPHdr.SetChecksum(0)
	replyIPHdr.SetChecksum(^replyIPHdr.CalculateChecksum())

	replyICMPHdr := header.ICMPv4(replyData)
	replyICMPHdr.SetType(header.ICMPv4EchoReply)
	replyICMPHdr.SetChecksum(0)
	replyICMPHdr.SetChecksum(^header.Checksum(replyData, 0))

	replyVV := buffer.View(replyIPHdr).ToVectorisedView()
	replyVV.AppendView(replyData)
	replyPkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
		ReserveHeaderBytes: header.IPv4MaximumHeaderSize,
		Data:               replyVV,
	})
	// defer replyPkt.DecRef()

	// Populate the network/transport headers in the packet buffer so the
	// ICMP packet goes through IPTables.
	if ok := parse.IPv4(replyPkt); !ok {
		panic("expected to parse IPv4 header we just created")
	}
	if ok := parse.ICMPv4(replyPkt); !ok {
		panic("expected to parse ICMPv4 header we just created")
	}
	return replyPkt, replyVV
}
