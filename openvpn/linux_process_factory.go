// +build linux

package openvpn

import (
	"github.com/mysterium/node/openvpn/config"
	"github.com/mysterium/node/openvpn/management"
)

func CreateNewProcess(openvpnBinary string, config *config.GenericConfig, middlewares ...management.Middleware) Process {
	return NewLinuxProcess(openvpnBinary, config, middlewares...)
}
