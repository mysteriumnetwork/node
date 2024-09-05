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

package netstack_provider

import (
	"bytes"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/mysteriumnetwork/node/config"
	netstack "github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack"
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

	resp, err := client.Get("http://107.173.23.19:8080/test")
	if err != nil {
		t.Error(err)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return
	}
	log.Println("Reply:", string(body))

	res := strings.HasPrefix(string(body), "Hello,")
	ok := "success"
	if !res {
		ok = "failed"
	}
	log.Println("Test result:", ok)
	// dev.Down()
	// tun.Close()
}

func startServer(t *testing.T, privKey, pubClinet wgtypes.Key) {
	tun, _, _, err := CreateNetTUNWithStack(
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
	InitUserspaceShaper(nil)

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
	startServer(t, privKey1, privKey2.PublicKey())
	startClient(t, privKey2, privKey1.PublicKey())
}
