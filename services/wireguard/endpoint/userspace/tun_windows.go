//go:build windows

package userspace

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"

	"github.com/mysteriumnetwork/node/utils/cmdutil"
)

var validInterfaceName = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)

type nativeTun struct {
	*tun.NativeTun
}

func CreateTUN(name string, mtu int) (tun.Device, error) {
	tunDevice, err := tun.CreateTUN(name, mtu)
	if err != nil {
		return nil, err
	}
	native := &nativeTun{NativeTun: tunDevice.(*tun.NativeTun)}

	if tunDevice.Name() != name {
		if err := renameInterface(tunDevice.Name(), name); err != nil {
			native.Close()
			return nil, fmt.Errorf("failed to rename interface: %w", err)
		}
	}

	if mtu != 0 {
		if err := setMTU(name, mtu); err != nil {
			native.Close()
			return nil, fmt.Errorf("failed to set MTU: %w", err)
		}
	}

	return native, nil
}

func (tun *nativeTun) Flush() error {
	return nil
}

func (tun *nativeTun) MTU() (int, error) {
	return device.DefaultMTU, nil
}

func validateInterfaceName(name string) error {
	if !validInterfaceName.MatchString(name) {
		return fmt.Errorf("invalid interface name: %q", name)
	}
	if strings.ContainsAny(name, "\"&|;`$(){}[]<>#~!*?\\ ") {
		return fmt.Errorf("invalid interface name: %q", name)
	}
	return nil
}

func renameInterface(name, newname string) error {
	if err := validateInterfaceName(name); err != nil {
		return fmt.Errorf("cannot rename interface: %w", err)
	}
	if err := validateInterfaceName(newname); err != nil {
		return fmt.Errorf("cannot rename interface: %w", err)
	}
	return cmdutil.Exec("netsh", "interface", "set", "interface",
		fmt.Sprintf("name=%s", name), fmt.Sprintf("newname=%s", newname))
}

func setMTU(name string, mtu int) error {
	if err := validateInterfaceName(name); err != nil {
		return fmt.Errorf("cannot set MTU: %w", err)
	}
	return cmdutil.Exec("netsh", "interface", "ipv4", "set", "subinterface",
		name, fmt.Sprintf("mtu=%d", mtu), "store=persistent")
}

func destroyDevice(name string) error {
	return nil
}
