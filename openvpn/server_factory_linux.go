// +build linux

/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

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
