package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"net/netip"
	"os"
	"strings"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	netstack "github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack"
	netstack_provider "github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack-provider"
)

const (
	mtu = 1420
)

func startClient(priv, pubServ wgtypes.Key) {

	tun, tnet, err := netstack.CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr("192.168.4.100")},
		[]netip.Addr{netip.MustParseAddr("8.8.8.8")},
		mtu)
	if err != nil {
		log.Panic(err)
	}

	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelError, "C> "))

	wgConf := &bytes.Buffer{}
	wgConf.WriteString("private_key=" + hex.EncodeToString(priv[:]) + "\n")
	wgConf.WriteString("public_key=" + hex.EncodeToString(pubServ[:]) + "\n")
	wgConf.WriteString("allowed_ip=0.0.0.0/0\n")
	wgConf.WriteString("endpoint=127.0.0.1:58120\n")

	if err = dev.IpcSetOperation(wgConf); err != nil {
		log.Panicln(err)
	}
	if err = dev.Up(); err != nil {
		log.Panic(err)
	}

	client := http.Client{
		Transport: &http.Transport{
			DialContext: tnet.DialContext,
		},
	}

	resp, err := client.Get("http://107.173.23.19:8080/test")
	if err != nil {
		log.Panic(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
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

func server(privKey, pubClinet wgtypes.Key) {

	tun, _, _, err := netstack_provider.CreateNetTUNWithStack(
		[]netip.Addr{netip.MustParseAddr("192.168.4.1")},
		53,
		mtu,
	)
	if err != nil {
		log.Panic(err)
	}
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelError, "S> "))

	wgConf := &bytes.Buffer{}
	wgConf.WriteString("private_key=" + hex.EncodeToString(privKey[:]) + "\n")
	wgConf.WriteString("listen_port=58120\n")
	wgConf.WriteString("public_key=" + hex.EncodeToString(pubClinet[:]) + "\n")
	wgConf.WriteString("allowed_ip=0.0.0.0/0\n")

	if err = dev.IpcSetOperation(wgConf); err != nil {
		log.Panicln(err)
	}
	if err = dev.Up(); err != nil {
		log.Panicln(err)
	}
}

func main() {
	log.Default().SetFlags(0)
	log.Println("Test #6022/v1")

	privKey1, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		log.Panicf("generating private key: %v \n", err)
	}
	privKey2, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		log.Panicf("generating private key: %v \n", err)
	}
	server(privKey1, privKey2.PublicKey())
	startClient(privKey2, privKey1.PublicKey())

	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
