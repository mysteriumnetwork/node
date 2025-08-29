package socks5client_test

import (
    "bufio"
    "context"
    "encoding/binary"
    "io"
    "net"
    "net/netip"
    "testing"
    "time"

    sc "github.com/mysteriumnetwork/node/services/wireguard/endpoint/socks5client"
)

type osAdapter struct{}

func (a *osAdapter) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
    var d net.Dialer
    return d.DialContext(ctx, network, address)
}

func (a *osAdapter) DialUDPAddrPort(laddr, raddr netip.AddrPort) (sc.UDPConn, error) {
    var la *net.UDPAddr
    if laddr.IsValid() || laddr.Port() != 0 {
        la = &net.UDPAddr{IP: net.IP(laddr.Addr().AsSlice()), Port: int(laddr.Port())}
    }
    ra := &net.UDPAddr{IP: net.IP(raddr.Addr().AsSlice()), Port: int(raddr.Port())}
    return net.DialUDP("udp", la, ra)
}

func (a *osAdapter) LookupContextHost(ctx context.Context, host string) ([]string, error) {
    return net.DefaultResolver.LookupHost(ctx, host)
}

func startSOCKS(t *testing.T, d sc.Dialer) (addr string, closeFn func()) {
    srv := &sc.Server{Dialer: d}
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("listen: %v", err)
    }
    addr = ln.Addr().String()
    _ = ln.Close()

    done := make(chan struct{})
    go func() {
        _ = srv.Serve(addr)
        close(done)
    }()
    // wait a bit for listener
    time.Sleep(50 * time.Millisecond)
    return addr, func() { srv.Close(); <-done }
}

func TestSOCKS5_CONNECT_Echo(t *testing.T) {
    // Start OS echo server
    l, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("ListenTCP: %v", err)
    }
    defer l.Close()
    tcpAddr := l.Addr().String()

    go func() {
        for {
            c, err := l.Accept()
            if err != nil {
                return
            }
            go func(cc net.Conn) {
                defer cc.Close()
                io.Copy(cc, cc)
            }(c)
        }
    }()

    // Start SOCKS server (using OS adapter)
    addr, stop := startSOCKS(t, &osAdapter{})
    defer stop()

    // Dial SOCKS
    c, err := net.Dial("tcp", addr)
    if err != nil {
        t.Fatalf("dial socks: %v", err)
    }
    defer c.Close()
    br := bufio.NewReader(c)
    bw := bufio.NewWriter(c)

    // Greeting
    bw.Write([]byte{0x05, 0x01, 0x00})
    bw.Flush()
    if b, _ := br.ReadByte(); b != 0x05 {
        t.Fatalf("bad greet ver")
    }
    if b, _ := br.ReadByte(); b != 0x00 {
        t.Fatalf("bad greet method")
    }

    // CONNECT to echo server
    host, portStr, _ := net.SplitHostPort(tcpAddr)
    ip := net.ParseIP(host)
    port, _ := net.LookupPort("tcp", portStr)
    pkt := []byte{0x05, 0x01, 0x00}
    if ip.To4() != nil {
        pkt = append(pkt, 0x01)
        pkt = append(pkt, ip.To4()...)
    } else {
        t.Fatalf("expect ipv4 addr for test")
    }
    p := make([]byte, 2)
    binary.BigEndian.PutUint16(p, uint16(port))
    pkt = append(pkt, p...)
    bw.Write(pkt)
    bw.Flush()
    // Read reply
    rep := make([]byte, 10)
    if _, err := io.ReadFull(br, rep[:4]); err != nil {
        t.Fatalf("read rep hdr: %v", err)
    }
    if rep[1] != 0x00 {
        t.Fatalf("rep not success: %d", rep[1])
    }
    // Send payload, expect echo
    msg := []byte("hello")
    if _, err := c.Write(msg); err != nil {
        t.Fatalf("write payload: %v", err)
    }
    buf := make([]byte, len(msg))
    if _, err := io.ReadFull(c, buf); err != nil {
        t.Fatalf("read echo: %v", err)
    }
    if string(buf) != string(msg) {
        t.Fatalf("echo mismatch: %q != %q", buf, msg)
    }
}

func TestSOCKS5_UDP_Associate(t *testing.T) {
    // Start UDP echo server (OS)
    u, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
    if err != nil {
        t.Fatalf("ListenUDP: %v", err)
    }
    defer u.Close()
    udpAddr := u.LocalAddr().(*net.UDPAddr)

    go func() {
        buf := make([]byte, 65535)
        for {
            n, addr, err := u.ReadFrom(buf)
            if err != nil {
                return
            }
            u.WriteTo(buf[:n], addr)
        }
    }()

    // Start SOCKS server
    addr, stop := startSOCKS(t, &osAdapter{})
    defer stop()

    // Control TCP
    c, err := net.Dial("tcp", addr)
    if err != nil {
        t.Fatalf("dial socks: %v", err)
    }
    defer c.Close()
    br := bufio.NewReader(c)
    bw := bufio.NewWriter(c)
    // greet
    bw.Write([]byte{0x05, 0x01, 0x00})
    bw.Flush()
    br.ReadByte()
    br.ReadByte()

    // UDP ASSOCIATE
    pkt := []byte{0x05, 0x03, 0x00, 0x01, 0, 0, 0, 0, 0, 0}
    bw.Write(pkt)
    bw.Flush()
    rep := make([]byte, 10)
    if _, err := io.ReadFull(br, rep); err != nil {
        t.Fatalf("udp rep: %v", err)
    }
    if rep[1] != 0x00 || rep[3] != 0x01 {
        t.Fatalf("bad rep")
    }
    bindIP := net.IP(rep[4:8])
    bindPort := int(binary.BigEndian.Uint16(rep[8:10]))

    // Send UDP packet with SOCKS5 UDP header
    uc, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: bindIP, Port: bindPort})
    if err != nil {
        t.Fatalf("dial udp: %v", err)
    }
    defer uc.Close()

    // Build header to UDP echo server
    hdr := []byte{0x00, 0x00, 0x00, 0x01}
    hdr = append(hdr, udpAddr.IP.To4()...)
    p := make([]byte, 2)
    binary.BigEndian.PutUint16(p, uint16(udpAddr.Port))
    hdr = append(hdr, p...)
    payload := []byte("ping")
    uc.Write(append(hdr, payload...))

    // Read response
    buf := make([]byte, 65535)
    uc.SetReadDeadline(time.Now().Add(time.Second))
    n, _, err := uc.ReadFrom(buf)
    if err != nil {
        t.Fatalf("udp read: %v", err)
    }
    // Strip header
    if n < 10 { // minimal header
        t.Fatalf("short udp resp")
    }
    // header is RSV RSV FRAG ATYP (IPv4) + addr(4) + port(2)
    data := buf[10:n]
    if string(data) != "ping" {
        t.Fatalf("udp echo mismatch: %q", string(data))
    }
}
