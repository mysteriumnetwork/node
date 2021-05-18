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

package wgcfg

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Stats represents wireguard peer statistics information.
type Stats struct {
	BytesSent     uint64    `json:"bytes_sent"`
	BytesReceived uint64    `json:"bytes_received"`
	LastHandshake time.Time `json:"last_handshake"`
}

// DeviceConfig describes wireguard device configuration.
type DeviceConfig struct {
	IfaceName  string    `json:"iface_name"`
	Subnet     net.IPNet `json:"subnet"`
	PrivateKey string    `json:"private_key"`
	ListenPort int       `json:"listen_port"`
	DNS        []string  `json:"dns"`
	// Used only for unix.
	DNSScriptDir string `json:"dns_script_dir"`

	Peer         Peer `json:"peer"`
	ReplacePeers bool `json:"replace_peers"`
}

// MarshalJSON implements json.Marshaler interface to provide human readable configuration.
func (dc DeviceConfig) MarshalJSON() ([]byte, error) {
	type peer struct {
		PublicKey              string   `json:"public_key"`
		Endpoint               string   `json:"endpoint"`
		AllowedIPs             []string `json:"allowed_i_ps"`
		KeepAlivePeriodSeconds int      `json:"keep_alive_period_seconds"`
	}

	type deviceConfig struct {
		IfaceName    string   `json:"iface_name"`
		Subnet       string   `json:"subnet"`
		PrivateKey   string   `json:"private_key"`
		ListenPort   int      `json:"listen_port"`
		DNS          []string `json:"dns"`
		DNSScriptDir string   `json:"dns_script_dir"`
		Peer         peer     `json:"peer"`
		ReplacePeers bool     `json:"replace_peers"`
	}

	var peerEndpoint string
	if dc.Peer.Endpoint != nil {
		peerEndpoint = dc.Peer.Endpoint.String()
	}

	return json.Marshal(&deviceConfig{
		IfaceName:    dc.IfaceName,
		Subnet:       dc.Subnet.String(),
		PrivateKey:   dc.PrivateKey,
		ListenPort:   dc.ListenPort,
		DNS:          dc.DNS,
		DNSScriptDir: dc.DNSScriptDir,
		Peer: peer{
			PublicKey:              dc.Peer.PublicKey,
			Endpoint:               peerEndpoint,
			AllowedIPs:             dc.Peer.AllowedIPs,
			KeepAlivePeriodSeconds: dc.Peer.KeepAlivePeriodSeconds,
		},
		ReplacePeers: dc.ReplacePeers,
	})
}

// UnmarshalJSON implements json.Unmarshaler interface to receive human readable configuration.
func (dc *DeviceConfig) UnmarshalJSON(data []byte) error {
	type peer struct {
		PublicKey              string   `json:"public_key"`
		Endpoint               string   `json:"endpoint"`
		AllowedIPs             []string `json:"allowed_i_ps"`
		KeepAlivePeriodSeconds int      `json:"keep_alive_period_seconds"`
	}

	type deviceConfig struct {
		IfaceName    string   `json:"iface_name"`
		Subnet       string   `json:"subnet"`
		PrivateKey   string   `json:"private_key"`
		ListenPort   int      `json:"listen_port"`
		DNS          []string `json:"dns"`
		DNSScriptDir string   `json:"dns_script_dir"`
		Peer         peer     `json:"peer"`
	}

	cfg := deviceConfig{}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("could not unmarshal device config: %w", err)
	}

	ip, ipnet, err := net.ParseCIDR(cfg.Subnet)
	if err != nil {
		return fmt.Errorf("could not parse subnet: %w", err)
	}

	var peerEndpoint *net.UDPAddr
	if cfg.Peer.Endpoint != "" {
		peerEndpoint, err = net.ResolveUDPAddr("udp", cfg.Peer.Endpoint)
		if err != nil {
			return fmt.Errorf("could not resolve peer endpoint: %w", err)
		}
	}

	dc.IfaceName = cfg.IfaceName
	dc.Subnet = *ipnet
	dc.Subnet.IP = ip
	dc.PrivateKey = cfg.PrivateKey
	dc.ListenPort = cfg.ListenPort
	dc.DNS = cfg.DNS
	dc.DNSScriptDir = cfg.DNSScriptDir
	dc.Peer = Peer{
		PublicKey:              cfg.Peer.PublicKey,
		Endpoint:               peerEndpoint,
		AllowedIPs:             cfg.Peer.AllowedIPs,
		KeepAlivePeriodSeconds: cfg.Peer.KeepAlivePeriodSeconds,
	}

	return nil
}

// Encode encodes device config into string representation which is used for
// userspace and kernel space wireguard configuration.
func (dc *DeviceConfig) Encode() string {
	var res strings.Builder
	keyBytes, err := base64.StdEncoding.DecodeString(dc.PrivateKey)
	if err != nil {
		log.Err(err).Msg("Could not decode device private key. Will use empty config.")
		return ""
	}
	hexKey := hex.EncodeToString(keyBytes)

	res.WriteString(fmt.Sprintf("private_key=%s\n", hexKey))
	res.WriteString(fmt.Sprintf("listen_port=%d\n", dc.ListenPort))
	res.WriteString(fmt.Sprintf("replace_peers=%t\n", dc.ReplacePeers))
	res.WriteString(dc.Peer.Encode())
	return res.String()
}

// Peer represents wireguard peer.
type Peer struct {
	PublicKey              string       `json:"public_key"`
	Endpoint               *net.UDPAddr `json:"endpoint"`
	AllowedIPs             []string     `json:"allowed_i_ps"`
	KeepAlivePeriodSeconds int          `json:"keep_alive_period_seconds"`
}

// Encode encodes device peer config into string representation which is used for
// userspace and kernel space wireguard configuration.
func (p *Peer) Encode() string {
	var res strings.Builder

	keyBytes, err := base64.StdEncoding.DecodeString(p.PublicKey)
	if err != nil {
		log.Err(err).Msg("Could not decode device public key. Will use empty config.")
		return ""
	}
	hexKey := hex.EncodeToString(keyBytes)
	res.WriteString(fmt.Sprintf("public_key=%s\n", hexKey))
	res.WriteString(fmt.Sprintf("persistent_keepalive_interval=%d\n", p.KeepAlivePeriodSeconds))
	if p.Endpoint != nil {
		res.WriteString(fmt.Sprintf("endpoint=%s\n", p.Endpoint.String()))
	}
	for _, ip := range p.AllowedIPs {
		res.WriteString(fmt.Sprintf("allowed_ip=%s\n", ip))
	}
	return res.String()
}
