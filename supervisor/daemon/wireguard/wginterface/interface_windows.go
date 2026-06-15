//go:build windows

package wginterface

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"

	"github.com/mysteriumnetwork/node/supervisor/daemon/wireguard/wginterface/firewall"
)

var validInterfaceName = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)

func validateInterfaceName(name string) error {
	if !validInterfaceName.MatchString(name) {
		return fmt.Errorf("invalid interface name: %q", name)
	}
	if strings.ContainsAny(name, "\"&|;`$(){}[]<>#~!*?\\") {
		return fmt.Errorf("invalid interface name: %q", name)
	}
	return nil
}

func createTunnel(interfaceName string, dns []string) (tunnel tun.Device, _ string, err error) {
	log.Info().Msg("Creating Wintun interface")
	if err := validateInterfaceName(interfaceName); err != nil {
		return nil, interfaceName, fmt.Errorf("could not create Wintun tunnel: %w", err)
	}
	wintun, err := tun.CreateTUN(interfaceName, device.DefaultMTU)
	if err != nil {
		return nil, interfaceName, fmt.Errorf("could not create Wintun tunnel: %w", err)
	}

	out, err := exec.Command("netsh", "interface", "ipv4", "set", "subinterface",
		interfaceName, fmt.Sprintf("mtu=%d", device.DefaultMTU), "store=persistent").CombinedOutput()
	if err != nil {
		return nil, interfaceName, fmt.Errorf("could not set MTU for tunnel: %w, %s", err, string(out))
	}

	nativeTun := wintun.(*tun.NativeTun)

	dnsIPs := []net.IP{}
	for _, d := range dns {
		dnsIPs = append(dnsIPs, net.ParseIP(d))
	}

	err = firewall.EnableFirewall(nativeTun.LUID(), false, dnsIPs)
	if err != nil {
		log.Warn().Err(err).Msg("Unable to enable DNS firewall rules")
	}

	wintunVersion, err := nativeTun.RunningVersion()
	if err != nil {
		log.Warn().Err(err).Msg("Unable to determine Wintun version")
	} else {
		log.Info().Msgf("Using Wintun/%s", wintunVersion)
	}
	return wintun, interfaceName, nil
}

func newUAPIListener(interfaceName string) (listener net.Listener, err error) {
	uapi, err := ipc.UAPIListen(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("could not listen for UAPI wg configuration: %w", err)
	}
	return uapi, nil
}

func applySocketPermissions(_ string, _ string) error {
	return nil
}

func disableFirewall() {
	firewall.DisableFirewall()
}
