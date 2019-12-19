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
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.zx2c4.com/wireguard/device"
)

// Example string source (with some slight modifications to use all fields):
// https://www.wireguard.com/xplatform/#example-dialog.
const okGet = `private_key=e84b5a6d2717c1003a13b431570353dbaca9146cf150c5f8575680feba52027a
listen_port=12912
fwmark=1
public_key=b85996fecc9c7f1fc6d2572a76eda11d59bcd20be8e543b15ce4bd85a8e75a33
preshared_key=188515093e952f5f22e865cef3012e72f8b5f0b598ac0309d5dacce3b70fcf52
allowed_ip=192.168.4.4/32
endpoint=[abcd:23::33%2]:51820
last_handshake_time_sec=1
last_handshake_time_nsec=2
public_key=58402e695ba1772b1cc9309755f043251ea77fdcf10fbe63989ceb7e19321376
tx_bytes=38333
rx_bytes=2224
allowed_ip=192.168.4.6/32
persistent_keepalive_interval=111
endpoint=182.122.22.19:3233
last_handshake_time_sec=0
last_handshake_time_nsec=0
public_key=662e14fd594556f522604703340351258903b64f35553763f19426ab2a515c58
endpoint=5.152.198.39:51820
last_handshake_time_sec=0
last_handshake_time_nsec=0
allowed_ip=192.168.4.10/32
allowed_ip=192.168.4.11/32
tx_bytes=1212111
rx_bytes=1929999999
protocol_version=1
errno=0

`

func TestParseDevice(t *testing.T) {
	// Used to trigger "parse peers" mode easily.
	const okKey = "public_key=0000000000000000000000000000000000000000000000000000000000000000\n"

	tests := []struct {
		name string
		res  []byte
		ok   bool
		d    *UserspaceDevice
	}{
		{
			name: "invalid key=value",
			res:  []byte("foo=bar=baz"),
		},
		{
			name: "invalid public_key",
			res:  []byte("public_key=xxx"),
		},
		{
			name: "short public_key",
			res:  []byte("public_key=abcd"),
		},
		{
			name: "invalid fwmark",
			res:  []byte("fwmark=foo"),
		},
		{
			name: "invalid endpoint",
			res:  []byte(okKey + "endpoint=foo"),
		},
		{
			name: "invalid allowed_ip",
			res:  []byte(okKey + "allowed_ip=foo"),
		},
		{
			name: "error",
			res:  []byte("errno=2\n\n"),
		},
		{
			name: "ok",
			res:  []byte(okGet),
			ok:   true,
			d: &UserspaceDevice{
				ListenPort:   12912,
				FirewallMark: 1,
				Peers: []UserspaceDevicePeer{
					{
						PublicKey: "uFmW/sycfx/G0lcqdu2hHVm80gvo5UOxXOS9hajnWjM=",
						Endpoint: &net.UDPAddr{
							IP:   net.ParseIP("abcd:23::33"),
							Port: 51820,
							Zone: "2",
						},
						LastHandshakeTime: time.Unix(1, 2),
						AllowedIPs: []net.IPNet{
							{
								IP:   net.IP{0xc0, 0xa8, 0x4, 0x4},
								Mask: net.IPMask{0xff, 0xff, 0xff, 0xff},
							},
						},
					},
					{
						PublicKey: "WEAuaVuhdyscyTCXVfBDJR6nf9zxD75jmJzrfhkyE3Y=",
						Endpoint: &net.UDPAddr{
							IP:   net.IPv4(182, 122, 22, 19),
							Port: 3233,
						},
						// Zero-value because UNIX timestamp of 0. Explicitly
						// set for documentation purposes here.
						LastHandshakeTime:           time.Time{},
						PersistentKeepaliveInterval: 111000000000,
						ReceiveBytes:                2224,
						TransmitBytes:               38333,
						AllowedIPs: []net.IPNet{
							{
								IP:   net.IP{0xc0, 0xa8, 0x4, 0x6},
								Mask: net.IPMask{0xff, 0xff, 0xff, 0xff},
							},
						},
					},
					{
						PublicKey: "Zi4U/VlFVvUiYEcDNANRJYkDtk81VTdj8ZQmqypRXFg=",
						Endpoint: &net.UDPAddr{
							IP:   net.IPv4(5, 152, 198, 39),
							Port: 51820,
						},
						ReceiveBytes:  1929999999,
						TransmitBytes: 1212111,
						AllowedIPs: []net.IPNet{
							{
								IP:   net.IP{0xc0, 0xa8, 0x4, 0xa},
								Mask: net.IPMask{0xff, 0xff, 0xff, 0xff},
							},
							{
								IP:   net.IP{0xc0, 0xa8, 0x4, 0xb},
								Mask: net.IPMask{0xff, 0xff, 0xff, 0xff},
							},
						},
						ProtocolVersion: 1,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ParseUserspaceDevice(func(w *bufio.Writer) *device.IPCError {
				_, _ = w.Write(tt.res)
				return nil
			})

			if tt.ok && err != nil {
				t.Fatalf("failed to get devices: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("expected an error, but none occurred")
			}
			if err != nil {
				return
			}

			assert.Equal(t, tt.d, d)
		})
	}
}
