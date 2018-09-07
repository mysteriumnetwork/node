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
	"sync"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/openvpn/config"
	"github.com/mysteriumnetwork/node/openvpn/linux"
	"github.com/mysteriumnetwork/node/openvpn/management"
)

const linuxProcess = "[Linux openvpn process] "

type linuxOpenvpnProcess struct {
	config  *config.GenericConfig
	process *openvpnProcess

	//runtime variables
	tunService   linux.TunnelService
	finished     *sync.WaitGroup
	processError error
}

func (ls *linuxOpenvpnProcess) Start() error {
	tunDevice, err := linux.FindFreeTunDevice()
	if err != nil {
		return err
	}

	ls.config.SetDevice(tunDevice.Name)
	ls.tunService = linux.NewLinuxTunnelService(
		tunDevice,
		ls.config.GetFullScriptPath(config.SimplePath("prepare-env.sh")),
	)

	err = ls.tunService.Start()
	if err != nil {
		return err
	}

	err = ls.process.Start()
	if err != nil {
		ls.tunService.Stop()
		return err
	}
	ls.finished.Add(1)
	go func() {
		ls.processError = ls.process.Wait()
		ls.tunService.Stop()
		log.Info(linuxProcess, "Process stopped, tun device removed")
		ls.finished.Done()
	}()
	return nil
}

func (ls *linuxOpenvpnProcess) Wait() error {
	ls.finished.Wait()
	log.Info(linuxProcess, "Wait finished")
	return ls.processError
}

func (ls *linuxOpenvpnProcess) Stop() {
	log.Info(linuxProcess, "Stop requested")

	if ls.tunService != nil {
		ls.tunService.Stop()
	}

	ls.process.Stop()
}

// NewLinuxProcess creates linux OS customized openvpn process
func NewLinuxProcess(
	openvpnBinary string,
	configuration *config.GenericConfig,
	middlewares ...management.Middleware,
) *linuxOpenvpnProcess {
	configuration.SetPersistTun()
	configuration.SetScriptParam("iproute", config.SimplePath("nonpriv-ip"))

	return &linuxOpenvpnProcess{
		config:   configuration,
		process:  newProcess(openvpnBinary, configuration, middlewares...),
		finished: &sync.WaitGroup{},
	}
}
