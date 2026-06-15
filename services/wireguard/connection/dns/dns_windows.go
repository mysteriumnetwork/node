//go:build windows

package dns

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

func validateInterfaceName(name string) error {
	if strings.ContainsAny(name, "\"&|;`$(){}[]<>#~!*?\\") {
		return fmt.Errorf("invalid interface name: %s", name)
	}
	return nil
}

func setDNS(cfg Config) error {
	if err := validateInterfaceName(cfg.IfaceName); err != nil {
		return fmt.Errorf("could not configure DNS: %w", err)
	}
	dnsIP := net.ParseIP(cfg.DNS[0])
	if dnsIP == nil {
		return fmt.Errorf("could not configure DNS: invalid DNS IP address: %s", cfg.DNS[0])
	}
	out, err := exec.Command("netsh", "interface", "ipv4", "set", "dnsservers",
		fmt.Sprintf("name=%s", cfg.IfaceName), "source=static",
		fmt.Sprintf("address=%s", dnsIP.String()), "validate=no").CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not configure DNS, %s:%v", string(out), err)
	}
	return nil
}

func cleanDNS(cfg Config) error {
	if err := validateInterfaceName(cfg.IfaceName); err != nil {
		return fmt.Errorf("could not clean DNS: %w", err)
	}
	out, err := exec.Command("netsh", "interface", "ipv4", "set", "dnsservers",
		fmt.Sprintf("name=%s", cfg.IfaceName), "source=static", "address=none",
		"validate=no", "register=both").CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not clean DNS, %s:%w", string(out), err)
	}
	return nil
}
