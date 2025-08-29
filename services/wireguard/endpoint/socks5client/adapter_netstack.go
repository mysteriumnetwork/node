package socks5client

import (
    "context"
    "net"
    "net/netip"

    netstackpkg "github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack"
)

type netstackAdapter struct{ t *netstackpkg.Net }

// NewNetstackAdapter exposes adapter for external users (e.g., dualproxyclient).
func NewNetstackAdapter(t *netstackpkg.Net) *netstackAdapter { return &netstackAdapter{t: t} }

func (a *netstackAdapter) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
    return a.t.DialContext(ctx, network, address)
}

func (a *netstackAdapter) DialUDPAddrPort(laddr, raddr netip.AddrPort) (UDPConn, error) {
    return a.t.DialUDPAddrPort(laddr, raddr)
}

func (a *netstackAdapter) LookupContextHost(ctx context.Context, host string) ([]string, error) {
    return a.t.LookupContextHost(ctx, host)
}
