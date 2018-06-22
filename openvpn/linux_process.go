package openvpn

import (
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/openvpn/config"
	"github.com/mysterium/node/openvpn/linux"
	"github.com/mysterium/node/openvpn/management"
	"sync"
)

const linuxProcess = "[Linux openvpn process] "

type linuxOpenvpnProcess struct {
	Process
	tunService linux.TunnelService
	//runtime variables
	finished     *sync.WaitGroup
	processError error
}

func (ls *linuxOpenvpnProcess) Start() error {
	//TODO - can we avoid hardcoding device name?
	//i.e. search for first available which is down (loop from 0 to x for fixed iteration count)
	ls.tunService.Add(linux.TunnelDevice{"tun0"})

	if err := ls.tunService.Start(); err != nil {
		return err
	}

	err := ls.Process.Start()
	if err != nil {
		ls.tunService.Stop()
		return err
	}
	ls.finished.Add(1)
	go func() {
		ls.processError = ls.Process.Wait()
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
	ls.Process.Stop()
}

func NewLinuxProcess(openvpnBinary string, config *config.GenericConfig, middlewares ...management.Middleware) Process {
	config.SetPersistTun()
	config.SetDevice("tun0")
	config.SetScriptParam("iproute", "nonpriv-ip", false)

	return &linuxOpenvpnProcess{
		Process:    newProcess(openvpnBinary, config, middlewares...),
		tunService: linux.NewLinuxTunnelService(),
		finished:   &sync.WaitGroup{},
	}
}
