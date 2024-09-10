/*
 * Copyright (C) 2024 The "MysteriumNetwork/node" Authors.
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

package shaper

import (
	"bytes"
	"encoding/hex"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/mysteriumnetwork/node/config"
	netstack "github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack"
	netstack_provider "github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack-provider"
)

func startClient(t *testing.T, priv, pubServ wgtypes.Key) {
	tun, tnet, err := netstack.CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr("192.168.4.100")},
		[]netip.Addr{netip.MustParseAddr("8.8.8.8")},
		device.DefaultMTU)
	if err != nil {
		t.Error(err)
		return
	}

	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelError, "C> "))

	wgConf := &bytes.Buffer{}
	wgConf.WriteString("private_key=" + hex.EncodeToString(priv[:]) + "\n")
	wgConf.WriteString("public_key=" + hex.EncodeToString(pubServ[:]) + "\n")
	wgConf.WriteString("allowed_ip=0.0.0.0/0\n")
	wgConf.WriteString("endpoint=127.0.0.1:58120\n")

	if err = dev.IpcSetOperation(wgConf); err != nil {
		t.Error(err)
		return
	}
	if err = dev.Up(); err != nil {
		t.Error(err)
		return
	}

	client := http.Client{
		Transport: &http.Transport{
			DialContext: tnet.DialContext,
		},
	}

	// resolve docker container hostname
	u, _ := url.Parse("http://shaper-websvc:8083/test")
	address, err := net.LookupHost(u.Hostname())
	if err != nil {
		t.Error(err)
		return
	}
	u.Host = address[0] + ":" + u.Port()

	resp, err := client.Get(u.String())
	if err != nil {
		t.Error(err)
		log.Println(err)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		log.Println(err)
		return
	}
	log.Println("Reply:", string(body))

	res := strings.HasPrefix(string(body), "Hello,")
	ok := "success"
	if !res {
		ok = "failed"
	}
	log.Println("Test result:", ok)
}

func startServer(t *testing.T, privKey, pubClinet wgtypes.Key) {
	tun, _, _, err := netstack_provider.CreateNetTUNWithStack(
		[]netip.Addr{netip.MustParseAddr("192.168.4.1")},
		53,
		device.DefaultMTU,
	)
	if err != nil {
		t.Error(err)
		return
	}
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelError, "S> "))

	wgConf := &bytes.Buffer{}
	wgConf.WriteString("private_key=" + hex.EncodeToString(privKey[:]) + "\n")
	wgConf.WriteString("listen_port=58120\n")
	wgConf.WriteString("public_key=" + hex.EncodeToString(pubClinet[:]) + "\n")
	wgConf.WriteString("allowed_ip=0.0.0.0/0\n")

	if err = dev.IpcSetOperation(wgConf); err != nil {
		t.Error(err)
		return
	}
	if err = dev.Up(); err != nil {
		t.Error(err)
		return
	}
}

func TestShaperEnabled(t *testing.T) {
	log.Default().SetFlags(0)

	config.Current.SetDefault(config.FlagShaperBandwidth.Name, "6250")
	config.Current.SetDefault(config.FlagShaperEnabled.Name, "true")
	config.FlagFirewallProtectedNetworks.Value = "10.0.0.0/8,127.0.0.0/8" // 192.168.0.0/16,

	netstack_provider.InitUserspaceShaper(nil)

	privKey1, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Error(err)
		return
	}
	privKey2, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Error(err)
		return
	}
	_, _ = privKey1, privKey2

	startServer(t, privKey1, privKey2.PublicKey())
	startClient(t, privKey2, privKey1.PublicKey())
}
