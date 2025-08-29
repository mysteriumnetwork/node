/*
 * Copyright (C) 2025 The "MysteriumNetwork/node" Authors.
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

package socks5client

import (
    "bufio"
    "context"
    "encoding/binary"
    "fmt"
    "io"
    "net"
    "net/netip"
    "sync"
    "time"
)

// UDPConn is a minimal interface for a UDP connection used by the server.
type UDPConn interface {
    Read([]byte) (int, error)
    Write([]byte) (int, error)
    SetReadDeadline(time.Time) error
    Close() error
}

// Dialer is the minimal interface server needs for TCP and UDP via an underlying network implementation.
type Dialer interface {
    DialContext(ctx context.Context, network, address string) (net.Conn, error)
    DialUDPAddrPort(laddr, raddr netip.AddrPort) (UDPConn, error)
    LookupContextHost(ctx context.Context, host string) ([]string, error)
}

// Server implements a minimal SOCKS5 server (no-auth) with TCP CONNECT and UDP ASSOCIATE.
type Server struct {
    Dialer Dialer

    ln     net.Listener
    closed chan struct{}
    once   sync.Once
}

func (s *Server) Serve(addr string) error {
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return err
    }
    s.ln = ln
    s.closed = make(chan struct{})

    for {
        conn, err := ln.Accept()
        if err != nil {
            select {
            case <-s.closed:
                return nil
            default:
            }
            if ne, ok := err.(net.Error); ok && ne.Temporary() {
                time.Sleep(50 * time.Millisecond)
                continue
            }
            return err
        }
        go s.serveConn(conn)
    }
}

func (s *Server) Close() {
    s.once.Do(func() {
        close(s.closed)
        if s.ln != nil {
            _ = s.ln.Close()
        }
    })
}

const (
    socksVer5        = 0x05
    socksMethodNoAuth = 0x00

    socksCmdConnect  = 0x01
    socksCmdBind     = 0x02 // not implemented
    socksCmdUDP      = 0x03

    socksAtypIPv4 = 0x01
    socksAtypFQDN = 0x03
    socksAtypIPv6 = 0x04

    socksRepSuccess              = 0x00
    socksRepGeneralFailure       = 0x01
    socksRepCommandNotSupported  = 0x07
    socksRepAddressTypeNotSupport = 0x08
)

func (s *Server) serveConn(c net.Conn) {
    defer c.Close()
    br := bufio.NewReader(c)
    bw := bufio.NewWriter(c)

    // Greeting
    ver, err := br.ReadByte()
    if err != nil || ver != socksVer5 {
        return
    }
    nMethods, err := br.ReadByte()
    if err != nil {
        return
    }
    methods := make([]byte, int(nMethods))
    if _, err := io.ReadFull(br, methods); err != nil {
        return
    }
    // Always select no-auth
    _, _ = bw.Write([]byte{socksVer5, socksMethodNoAuth})
    if err := bw.Flush(); err != nil {
        return
    }

    // Request
    hdr := make([]byte, 4)
    if _, err := io.ReadFull(br, hdr); err != nil {
        return
    }
    if hdr[0] != socksVer5 {
        return
    }
    cmd := hdr[1]
    atyp := hdr[3]

    var dstHost string
    var dstPort uint16
    switch atyp {
    case socksAtypIPv4:
        addr := make([]byte, 4)
        if _, err := io.ReadFull(br, addr); err != nil {
            return
        }
        port := make([]byte, 2)
        if _, err := io.ReadFull(br, port); err != nil {
            return
        }
        dstHost = net.IP(addr).String()
        dstPort = binary.BigEndian.Uint16(port)
    case socksAtypIPv6:
        addr := make([]byte, 16)
        if _, err := io.ReadFull(br, addr); err != nil {
            return
        }
        port := make([]byte, 2)
        if _, err := io.ReadFull(br, port); err != nil {
            return
        }
        dstHost = net.IP(addr).String()
        dstPort = binary.BigEndian.Uint16(port)
    case socksAtypFQDN:
        ln, err := br.ReadByte()
        if err != nil {
            return
        }
        name := make([]byte, int(ln))
        if _, err := io.ReadFull(br, name); err != nil {
            return
        }
        port := make([]byte, 2)
        if _, err := io.ReadFull(br, port); err != nil {
            return
        }
        dstHost = string(name)
        dstPort = binary.BigEndian.Uint16(port)
    default:
        s.writeReply(bw, socksRepAddressTypeNotSupport, net.IPv4zero, 0)
        return
    }

    switch cmd {
    case socksCmdConnect:
        s.handleConnect(c, br, bw, dstHost, int(dstPort))
    case socksCmdUDP:
        s.handleUDP(c, br, bw)
    default:
        s.writeReply(bw, socksRepCommandNotSupported, net.IPv4zero, 0)
    }
}

func (s *Server) writeReply(bw *bufio.Writer, rep byte, bindAddr net.IP, bindPort int) {
    // We always reply with IPv4 form (ATYP=1) for simplicity.
    pkt := make([]byte, 0, 10)
    pkt = append(pkt, socksVer5, rep, 0x00, socksAtypIPv4)
    if bindAddr == nil || bindAddr.To4() == nil {
        pkt = append(pkt, []byte{0, 0, 0, 0}...)
    } else {
        pkt = append(pkt, bindAddr.To4()...)
    }
    p := make([]byte, 2)
    binary.BigEndian.PutUint16(p, uint16(bindPort))
    pkt = append(pkt, p...)
    _, _ = bw.Write(pkt)
    _ = bw.Flush()
}

func (s *Server) handleConnect(c net.Conn, br *bufio.Reader, bw *bufio.Writer, host string, port int) {
    ctx := context.Background()
    rc, err := s.Dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
    if err != nil {
        s.writeReply(bw, socksRepGeneralFailure, net.IPv4zero, 0)
        return
    }
    defer rc.Close()

    // Success
    s.writeReply(bw, socksRepSuccess, net.IPv4zero, 0)

    // Bidirectional copy
    proxyStream(c, rc)
}

func (s *Server) handleUDP(c net.Conn, br *bufio.Reader, bw *bufio.Writer) {
    // Bind a local UDP socket for client datagrams.
    // Limit exposure to loopback.
    // no need to build a netip.Addr for OS listener
    // Use OS UDP for client-side (so the app can send UDP here), but for simplicity
    // use the stdlib net since we only need a local socket.
    uc, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
    if err != nil {
        s.writeReply(bw, socksRepGeneralFailure, net.IPv4zero, 0)
        return
    }

    // Reply success with bound address/port for client to send UDP packets to.
    bind := uc.LocalAddr().(*net.UDPAddr)
    s.writeReply(bw, socksRepSuccess, bind.IP, bind.Port)

    done := make(chan struct{})
    go func() {
        defer close(done)
        buf := make([]byte, 1<<16)
        for {
            _ = uc.SetReadDeadline(time.Now().Add(2 * time.Minute))
            n, clientAddr, err := uc.ReadFromUDP(buf)
            if err != nil {
                if ne, ok := err.(net.Error); ok && ne.Timeout() {
                    return
                }
                return
            }
            if n < 4+2 { // minimal header
                continue
            }
            // Parse UDP header (RSV, RSV, FRAG)
            if buf[2] != 0x00 { // FRAG not supported
                continue
            }
            atype := buf[3]
            off := 4
            var rIP net.IP
            var rPort uint16
            switch atype {
            case socksAtypIPv4:
                if n < off+4+2 {
                    continue
                }
                rIP = net.IP(buf[off : off+4])
                off += 4
            case socksAtypIPv6:
                if n < off+16+2 {
                    continue
                }
                rIP = net.IP(buf[off : off+16])
                off += 16
            case socksAtypFQDN:
                if n < off+1 {
                    continue
                }
                ln := int(buf[off])
                off++
                if n < off+ln+2 {
                    continue
                }
                name := string(buf[off : off+ln])
                off += ln
                // Resolve using tunnel DNS
                addrs, err := s.Dialer.LookupContextHost(context.Background(), name)
                if err != nil || len(addrs) == 0 {
                    continue
                }
                rIP = net.ParseIP(addrs[0])
            default:
                continue
            }
            rPort = binary.BigEndian.Uint16(buf[off : off+2])
            off += 2
            payload := buf[off:n]

            // Send to remote via tunnel
            rAddr, ok := netip.AddrFromSlice(rIP)
            if !ok {
                continue
            }
            // Connect for this peer and write packet
            pc, err := s.Dialer.DialUDPAddrPort(netip.AddrPort{}, netip.AddrPortFrom(rAddr, rPort))
            if err != nil {
                continue
            }
            _, _ = pc.Write(payload)

            // Collect immediate responses from the same remote socket
            _ = pc.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
            for {
                rb := make([]byte, 1<<16)
                rn, rerr := pc.Read(rb)
                if rerr != nil {
                    if ne, ok := rerr.(net.Error); ok && ne.Timeout() {
                        break
                    }
                    break
                }
                // Build SOCKS5 UDP response header. We don't know the src addr here from pc,
                // so set ATYP=IPv4 and ADDR=0.0.0.0:0 which clients typically ignore.
                hdr := []byte{0x00, 0x00, 0x00, socksAtypIPv4, 0, 0, 0, 0, 0, 0}
                out := append(hdr, rb[:rn]...)
                _, _ = uc.WriteToUDP(out, clientAddr)
            }
            _ = pc.Close()
        }
    }()

    // Keep TCP control connection open until client closes it; we don't expect more data here.
    // Any read will block; set a read deadline to clean up eventually.
    _ = c.SetReadDeadline(time.Now().Add(2 * time.Minute))
    _, _ = br.Peek(1)
    _ = uc.Close()
    <-done
}

func buildUDPResponseHeader(addr net.Addr) []byte {
    // RSV RSV FRAG ATYP DST.ADDR DST.PORT
    var atyp byte
    var ip net.IP
    var port int

    switch a := addr.(type) {
    case *net.UDPAddr:
        ip = a.IP
        port = a.Port
    default:
        // Fallback: 0.0.0.0:0
        ip = net.IPv4zero
        port = 0
    }
    if ip.To4() != nil {
        atyp = socksAtypIPv4
    } else if ip.To16() != nil {
        atyp = socksAtypIPv6
    } else {
        atyp = socksAtypIPv4
        ip = net.IPv4zero
        port = 0
    }
    b := []byte{0x00, 0x00, 0x00, atyp}
    if atyp == socksAtypIPv4 {
        b = append(b, ip.To4()...)
    } else {
        b = append(b, ip.To16()...)
    }
    p := make([]byte, 2)
    binary.BigEndian.PutUint16(p, uint16(port))
    b = append(b, p...)
    return b
}

func proxyStream(left, right net.Conn) {
    var wg sync.WaitGroup
    wg.Add(2)

    cpy := func(dst, src net.Conn) {
        defer wg.Done()
        _, _ = io.Copy(dst, src)
        _ = dst.Close()
    }
    go cpy(left, right)
    go cpy(right, left)
    wg.Wait()
}
