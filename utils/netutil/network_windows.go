//go:build windows

package netutil

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func validateInterfaceName(name string) error {
	if strings.ContainsAny(name, "\"&|;`$(){}[]<>#~!*?\\ ") {
		return fmt.Errorf("invalid interface name: %s", name)
	}
	return nil
}

func assignIP(iface string, subnet net.IPNet) error {
	if err := validateInterfaceName(iface); err != nil {
		return errors.Wrap(err, "invalid interface for assignIP")
	}
	out, err := exec.Command("netsh", "interface", "ip", "set", "address",
		fmt.Sprintf("name=%s", iface), "source=static",
		subnet.String()).CombinedOutput()
	return errors.Wrap(err, string(out))
}

func excludeRoute(ip, gw net.IP) error {
	out, err := exec.Command("route", "add",
		ip.String()+"/32", gw.String()).CombinedOutput()
	return errors.Wrap(err, string(out))
}

func deleteRoute(ip, gw string) error {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address for route deletion: %s", ip)
	}
	out, err := exec.Command("route", "delete",
		parsedIP.String()+"/32").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete route: %w, %s", err, string(out))
	}
	return nil
}

func addDefaultRoute(name string) error {
	id, gw, err := interfaceInfo(name)
	if err != nil {
		return errors.Wrap(err, "failed to get info of interface: "+name)
	}

	if out, err := exec.Command("route", "add", "0.0.0.0/1", gw, "if", id).CombinedOutput(); err != nil {
		return errors.Wrap(err, string(out))
	}

	if out, err := exec.Command("route", "add", "128.0.0.0/1", gw, "if", id).CombinedOutput(); err != nil {
		return errors.Wrap(err, string(out))
	}

	if out, err := exec.Command("route", "add", "::/1", "100::1", "if", id).CombinedOutput(); err != nil {
		return errors.Wrap(err, string(out))
	}

	if out, err := exec.Command("route", "add", "8000::/1", "100::1", "if", id).CombinedOutput(); err != nil {
		return errors.Wrap(err, string(out))
	}

	return nil
}

func interfaceInfo(name string) (id, gw string, err error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get interfaces "+name)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get interfaces addresses")
	}

	var ipv4 net.IP
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse an interface IP address")
		}

		if ip.To4() == nil {
			continue
		}

		if ipv4.Equal(net.IPv4zero) {
			return "", "", errors.New("failed to get interface info: exactly 1 IPv4 expected")
		}

		ipv4 = ip.To4()
		ipv4[net.IPv4len-1] = byte(1)
	}

	return strconv.Itoa(iface.Index), ipv4.String(), nil
}

func logNetworkStats() {
	for _, args := range []string{"ipconfig", "/all"} {
		out, err := exec.Command("ipconfig", "/all").CombinedOutput()
		logOutputToTrace(out, err, args)
	}
	{
		out, err := exec.Command("netstat", "-r").CombinedOutput()
		logOutputToTrace(out, err, "netstat -r")
	}
}
