//go:build !windows

package dns

import (
	"net"
	"os/exec"
	"path"
)

var defaultScriptDir = "/etc/mysterium-node"

func setDNS(cfg Config) error {
	dnsIP := net.ParseIP(cfg.DNS[0])
	if dnsIP == nil {
		return exec.Command("true").Run()
	}
	cmd := exec.Command(path.Join(defaultScriptDir, "update-resolv-conf"))
	cmd.Env = append(cmd.Environ(),
		"script_type=up",
		"dev="+cfg.IfaceName,
		"foreign_option_1=dhcp-option DNS "+dnsIP.String())
	return cmd.Run()
}

func cleanDNS(cfg Config) error {
	cmd := exec.Command(path.Join(defaultScriptDir, "update-resolv-conf"))
	cmd.Env = append(cmd.Environ(), "script_type=down", "dev="+cfg.IfaceName)
	return cmd.Run()
}


