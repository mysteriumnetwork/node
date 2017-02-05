package main

import (
	"github.com/mysterium/node/openvpn"
)

func main() {
	vpnClient := openvpn.NewClient("68.235.53.140", "pre-shared.key")
	vpnClient.Start()
}
