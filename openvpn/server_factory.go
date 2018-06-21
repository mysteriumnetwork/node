// +build !linux

package openvpn

import "github.com/mysterium/node/openvpn/management"

func CreateNewServer(openvpnBinary string, generateConfig ServerConfigGenerator, middlewares ...management.Middleware) Process {
	serverConfig := generateConfig()
	serverConfig.SetDevice("tun")
	return NewServer(openvpnBinary, func() *ServerConfig { return serverConfig }, middlewares...)
}
