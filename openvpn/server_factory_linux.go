// +build linux

package openvpn

import (
	"github.com/cihub/seelog"
	"github.com/mysterium/node/openvpn/management"
	"github.com/mysterium/node/openvpn/tun"
)

type linuxOpenvpnServer struct {
	Process
	tunService tun.Service
}

func CreateNewServer(openvpnBinary string, generateConfig ServerConfigGenerator, middlewares ...management.Middleware) Process {
	config := generateConfig()
	config.SetPersistTun()
	config.SetDevice("tun0")
	config.SetIpRouteScript("nonpriv-ip")

	return &linuxOpenvpnServer{
		Process:    NewServer(openvpnBinary, func() *ServerConfig { return config }, middlewares...),
		tunService: tun.NewLinuxTunnelService(),
	}
}

func (ls *linuxOpenvpnServer) Start() error {
	ls.tunService.Add(tun.Device{"tun0"})

	if err := ls.tunService.Start(); err != nil {
		return err
	}

	return ls.Process.Start()
}

func (ls *linuxOpenvpnServer) Wait() error {
	seelog.Info("Hook for Wait on linux!")
	return nil
}

func (ls *linuxOpenvpnServer) Stop() {
	ls.Process.Stop()
	ls.Process.Wait()
	ls.tunService.Stop()
}
