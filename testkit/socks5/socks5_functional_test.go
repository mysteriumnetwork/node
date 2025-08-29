package socks5_test

import (
    "bufio"
    "context"
    "encoding/binary"
    "fmt"
    "net"
    "os"
    "testing"
    "time"

    "golang.org/x/net/dns/dnsmessage"
)

// dialIfListening tries to connect to addr within timeout; returns conn or nil.
func dialIfListening(addr string, timeout time.Duration) net.Conn {
    d := net.Dialer{Timeout: timeout}
    c, err := d.DialContext(context.Background(), "tcp", addr)
    if err != nil {
        return nil
    }
    return c
}

// buildDNSQuery creates a simple A query for example.com.
func buildDNSQuery() []byte {
    b := dnsmessage.NewBuilder(nil, dnsmessage.Header{RecursionDesired: true})
    _ = b.StartQuestions()
    _ = b.Question(dnsmessage.Question{
        Name:  dnsmessage.MustNewName("example.com."),
        Type:  dnsmessage.TypeA,
        Class: dnsmessage.ClassINET,
    })
    msg, _ := b.Finish()
    return msg
}

// socks5 UDP header builder for IPv4 target.
func buildSocksUDPHeader(ip net.IP, port int) []byte {
    h := []byte{0, 0, 0, 1}
    h = append(h, ip.To4()...)
    p := make([]byte, 2)
    binary.BigEndian.PutUint16(p, uint16(port))
    h = append(h, p...)
    return h
}

func TestSOCKS5_UDP_DNS_Associate(t *testing.T) {
    // Allow override of socks and DNS via env.
    socksAddr := os.Getenv("SOCKS5_ADDR")
    if socksAddr == "" {
        socksAddr = "127.0.0.1:10001"
    }
    dnsHost := os.Getenv("DNS_HOST")
    if dnsHost == "" {
        dnsHost = "1.1.1.1"
    }
    dnsPort := 53

    // Skip if SOCKS not listening.
    c := dialIfListening(socksAddr, 500*time.Millisecond)
    if c == nil {
        if os.Getenv("SOCKS5_REQUIRED") != "" {
            t.Fatalf("SOCKS5 not listening at %s and SOCKS5_REQUIRED set", socksAddr)
        }
        t.Skipf("SOCKS5 not listening at %s; skipping functional test", socksAddr)
        return
    }
    defer c.Close()
    br := bufio.NewReader(c)
    bw := bufio.NewWriter(c)

    // Greeting: no auth
    bw.Write([]byte{0x05, 0x01, 0x00})
    bw.Flush()
    if v, _ := br.ReadByte(); v != 0x05 { t.Fatalf("bad ver") }
    if m, _ := br.ReadByte(); m != 0x00 { t.Fatalf("bad method") }

    // UDP ASSOCIATE with 0.0.0.0:0
    bw.Write([]byte{0x05, 0x03, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
    bw.Flush()
    rep := make([]byte, 10)
    if _, err := br.Read(rep); err != nil { t.Fatalf("udp rep: %v", err) }
    if rep[1] != 0x00 { t.Fatalf("rep=%d", rep[1]) }
    bindIP := net.IP(rep[4:8])
    bindPort := int(binary.BigEndian.Uint16(rep[8:10]))

    uc, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: bindIP, Port: bindPort})
    if err != nil { t.Fatalf("udp dial: %v", err) }
    defer uc.Close()

    // Send DNS query via UDP ASSOCIATE
    hdr := buildSocksUDPHeader(net.ParseIP(dnsHost), dnsPort)
    payload := buildDNSQuery()
    if _, err := uc.Write(append(hdr, payload...)); err != nil { t.Fatalf("udp write: %v", err) }

    // Read response and ensure it parses as DNS.
    buf := make([]byte, 4096)
    _ = uc.SetReadDeadline(time.Now().Add(2 * time.Second))
    n, _, err := uc.ReadFrom(buf)
    if err != nil { t.Fatalf("udp read: %v", err) }
    if n < 10 { t.Fatalf("short resp") }
    // Strip socks header (assume IPv4 header size = 10)
    dnsBytes := buf[10:n]
    var p dnsmessage.Parser
    if _, err := p.Start(dnsBytes); err != nil {
        t.Fatalf("dns parse: %v", err)
    }
}

func TestSOCKS5_TCP_CONNECT_HTTP(t *testing.T) {
    // Note: Requires SOCKS5 running locally (default 127.0.0.1:10001).
    socksAddr := os.Getenv("SOCKS5_ADDR")
    if socksAddr == "" {
        socksAddr = "127.0.0.1:10001"
    }
    targetHost := os.Getenv("TCP_HOST")
    if targetHost == "" {
        targetHost = "example.com"
    }
    targetPort := 80

    c := dialIfListening(socksAddr, 500*time.Millisecond)
    if c == nil {
        if os.Getenv("SOCKS5_REQUIRED") != "" {
            t.Fatalf("SOCKS5 not listening at %s and SOCKS5_REQUIRED set", socksAddr)
        }
        t.Skipf("SOCKS5 not listening at %s; skipping functional test", socksAddr)
        return
    }
    defer c.Close()
    br := bufio.NewReader(c)
    bw := bufio.NewWriter(c)

    // Greeting: no auth
    bw.Write([]byte{0x05, 0x01, 0x00})
    bw.Flush()
    if v, _ := br.ReadByte(); v != 0x05 { t.Fatalf("bad ver") }
    if m, _ := br.ReadByte(); m != 0x00 { t.Fatalf("bad method") }

    // CONNECT request using domain name (ATYP=3)
    hostBytes := []byte(targetHost)
    pkt := []byte{0x05, 0x01, 0x00, 0x03, byte(len(hostBytes))}
    pkt = append(pkt, hostBytes...)
    p := make([]byte, 2)
    binary.BigEndian.PutUint16(p, uint16(targetPort))
    pkt = append(pkt, p...)
    bw.Write(pkt)
    bw.Flush()

    // Read reply
    rep := make([]byte, 4)
    if _, err := br.Read(rep); err != nil { t.Fatalf("read rep hdr: %v", err) }
    if rep[1] != 0x00 { t.Fatalf("connect rep=%d", rep[1]) }
    // Consume BND.ADDR/BND.PORT (skip)
    atyp := rep[3]
    switch atyp {
    case 0x01:
        ioDiscardN(t, br, 4+2)
    case 0x04:
        ioDiscardN(t, br, 16+2)
    case 0x03:
        ln, _ := br.ReadByte()
        ioDiscardN(t, br, int(ln)+2)
    default:
        t.Fatalf("bad atyp in rep: %d", atyp)
    }

    // Send HTTP/1.1 request and read response
    req := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", targetHost)
    if _, err := bw.WriteString(req); err != nil { t.Fatalf("write req: %v", err) }
    if err := bw.Flush(); err != nil { t.Fatalf("flush: %v", err) }

    _ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
    buf := make([]byte, 8192)
    n, err := br.Read(buf)
    if err != nil {
        t.Fatalf("read resp: %v", err)
    }
    if n == 0 || string(buf[:8]) != "HTTP/1.1" {
        t.Fatalf("unexpected HTTP response: %q", string(buf[:min(n, 64)]))
    }
}

func ioDiscardN(t *testing.T, r *bufio.Reader, n int) {
    t.Helper()
    for n > 0 {
        k := n
        if k > 512 { k = 512 }
        if _, err := r.Discard(k); err != nil { t.Fatalf("discard: %v", err) }
        n -= k
    }
}

func min(a, b int) int { if a < b { return a }; return b }
