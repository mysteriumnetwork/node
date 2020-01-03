/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package wireguard

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// UserspaceDevice is a WireGuard device.
type UserspaceDevice struct {
	// ListenPort is the device's network listening port.
	ListenPort int

	// FirewallMark is the device's current firewall mark.
	//
	// The firewall mark can be used in conjunction with firewall software to
	// take action on outgoing WireGuard packets.
	FirewallMark int

	// Peers is the list of network peers associated with this device.
	Peers []UserspaceDevicePeer
}

// UserspaceDevicePeer is a WireGuard peer to a Device.
type UserspaceDevicePeer struct {
	// PublicKey is the public key of a peer, computed from its private key.
	//
	// PublicKey is always present in a Peer.
	PublicKey string

	// Endpoint is the most recent source address used for communication by
	// this Peer.
	Endpoint *net.UDPAddr

	// PersistentKeepaliveInterval specifies how often an "empty" packet is sent
	// to a peer to keep a connection alive.
	//
	// A value of 0 indicates that persistent keepalives are disabled.
	PersistentKeepaliveInterval time.Duration

	// LastHandshakeTime indicates the most recent time a handshake was performed
	// with this peer.
	//
	// A zero-value time.Time indicates that no handshake has taken place with
	// this peer.
	LastHandshakeTime time.Time

	// ReceiveBytes indicates the number of bytes received from this peer.
	ReceiveBytes int64

	// TransmitBytes indicates the number of bytes transmitted to this peer.
	TransmitBytes int64

	// AllowedIPs specifies which IPv4 and IPv6 addresses this peer is allowed
	// to communicate on.
	//
	// 0.0.0.0/0 indicates that all IPv4 addresses are allowed, and ::/0
	// indicates that all IPv6 addresses are allowed.
	AllowedIPs []net.IPNet

	// ProtocolVersion specifies which version of the WireGuard protocol is used
	// for this Peer.
	//
	// A value of 0 indicates that the most recent protocol version will be used.
	ProtocolVersion int
}

// ParseUserspaceDevice parses WireGuard device state buffer.
func ParseUserspaceDevice(ipcGetOp func(w *bufio.Writer) *device.IPCError) (*UserspaceDevice, error) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	if err := ipcGetOp(writer); err != nil {
		return nil, err
	}
	if err := writer.Flush(); err != nil {
		return nil, err
	}

	var dp deviceParser
	s := bufio.NewScanner(&buf)
	for s.Scan() {
		b := s.Bytes()
		if len(b) == 0 {
			// Empty line, done parsing.
			break
		}

		// All data is in key=value format.
		kvs := bytes.Split(b, []byte("="))
		if len(kvs) != 2 {
			return nil, fmt.Errorf("invalid key=value pair: %q", string(b))
		}

		dp.Parse(string(kvs[0]), string(kvs[1]))
	}

	if dp.err != nil {
		return nil, dp.err
	}
	return &dp.d, nil
}

// A deviceParser accumulates information about a Device and its Peers. Adapted from
// https://github.com/WireGuard/wgctrl-go/blob/master/internal/wguser/parse.go.
type deviceParser struct {
	d   UserspaceDevice
	err error

	parsePeers    bool
	peers         int
	hsSec, hsNano int
}

// Parse parses a single key/value pair into fields of a Device.
func (dp *deviceParser) Parse(key, value string) {
	switch key {
	case "errno":
		// 0 indicates success, anything else returns an error number that matches
		// definitions from errno.h.
		if errno := dp.parseInt(value); errno != 0 {
			dp.err = os.NewSyscallError("read", fmt.Errorf("wguser: errno=%d", errno))
			return
		}
	case "public_key":
		// We've either found the first peer or the next peer.  Stop parsing
		// Device fields and start parsing Peer fields, including the public
		// key indicated here.
		dp.parsePeers = true
		dp.peers++

		dp.d.Peers = append(dp.d.Peers, UserspaceDevicePeer{
			PublicKey: dp.parseKey(value),
		})
		return
	}

	// Are we parsing peer fields?
	if dp.parsePeers {
		dp.peerParse(key, value)
		return
	}

	// Device field parsing.
	switch key {
	case "listen_port":
		dp.d.ListenPort = dp.parseInt(value)
	case "fwmark":
		dp.d.FirewallMark = dp.parseInt(value)
	}
}

// curPeer returns the current Peer being parsed so its fields can be populated.
func (dp *deviceParser) curPeer() *UserspaceDevicePeer {
	return &dp.d.Peers[dp.peers-1]
}

// peerParse parses a key/value field into the current Peer.
func (dp *deviceParser) peerParse(key, value string) {
	p := dp.curPeer()
	switch key {
	case "endpoint":
		p.Endpoint = dp.parseAddr(value)
	case "last_handshake_time_sec":
		dp.hsSec = dp.parseInt(value)
	case "last_handshake_time_nsec":
		dp.hsNano = dp.parseInt(value)

		// Assume that we've seen both seconds and nanoseconds and populate this
		// field now. However, if both fields were set to 0, assume we have never
		// had a successful handshake with this peer, and return a zero-value
		// time.Time to our callers.
		if dp.hsSec > 0 && dp.hsNano > 0 {
			p.LastHandshakeTime = time.Unix(int64(dp.hsSec), int64(dp.hsNano))
		}
	case "tx_bytes":
		p.TransmitBytes = dp.parseInt64(value)
	case "rx_bytes":
		p.ReceiveBytes = dp.parseInt64(value)
	case "persistent_keepalive_interval":
		p.PersistentKeepaliveInterval = time.Duration(dp.parseInt(value)) * time.Second
	case "allowed_ip":
		cidr := dp.parseCIDR(value)
		if cidr != nil {
			p.AllowedIPs = append(p.AllowedIPs, *cidr)
		}
	case "protocol_version":
		p.ProtocolVersion = dp.parseInt(value)
	}
}

// parseKey parses a Key from a hex string.
func (dp *deviceParser) parseKey(s string) string {
	if dp.err != nil {
		return ""
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		dp.err = err
		return ""
	}

	key, err := wgtypes.NewKey(b)
	if err != nil {
		dp.err = err
		return ""
	}

	return key.String()
}

// parseInt parses an integer from a string.
func (dp *deviceParser) parseInt(s string) int {
	if dp.err != nil {
		return 0
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		dp.err = err
		return 0
	}

	return v
}

// parseInt64 parses an int64 from a string.
func (dp *deviceParser) parseInt64(s string) int64 {
	if dp.err != nil {
		return 0
	}

	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		dp.err = err
		return 0
	}

	return v
}

// parseAddr parses a UDP address from a string.
func (dp *deviceParser) parseAddr(s string) *net.UDPAddr {
	if dp.err != nil {
		return nil
	}

	addr, err := net.ResolveUDPAddr("udp", s)
	if err != nil {
		dp.err = err
		return nil
	}

	return addr
}

// parseInt parses an address CIDR from a string.
func (dp *deviceParser) parseCIDR(s string) *net.IPNet {
	if dp.err != nil {
		return nil
	}

	_, cidr, err := net.ParseCIDR(s)
	if err != nil {
		dp.err = err
		return nil
	}

	return cidr
}
