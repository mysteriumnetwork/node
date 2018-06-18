package openvpn

import (
	"errors"
	"github.com/mysterium/node/openvpn/config"
	"github.com/mysterium/node/openvpn/management"
	"sync"
	"time"
)

// Process defines openvpn process interface with basic controls
type Process interface {
	Start() error
	Wait() error
	Stop() error
}

type openvpnProcess struct {
	config     *config.GenericConfig
	management *management.Management
	cmd        *CmdWrapper
}

func (openvpn *openvpnProcess) Start() error {
	err := openvpn.management.WaitForConnection()
	if err != nil {
		return err
	}

	addr := openvpn.management.BoundAddress
	openvpn.config.SetManagementAddress(addr.IP, addr.Port)

	// Fetch the current arguments
	arguments, err := (*openvpn.config).ToArguments()
	if err != nil {
		return err
	}

	//nil returned from process.Start doesn't guarantee that openvpn itself initialized correctly and accepted all arguments
	//it simply means that OS started process with specified args
	err = openvpn.cmd.Start(arguments)
	if err != nil {
		openvpn.management.Stop()
		return err
	}

	select {
	case connAccepted := <-openvpn.management.Connected:
		if connAccepted {
			return nil
		}
		return errors.New("management failed to accept connection")
	case exitError := <-openvpn.cmd.CmdExitError:
		openvpn.management.Stop()
		if exitError != nil {
			return exitError
		}
		return errors.New("openvpn process died too early")
	case <-time.After(2 * time.Second):
		return errors.New("management connection wait timeout")
	}
}

func (openvpn *openvpnProcess) Wait() error {
	return openvpn.cmd.Wait()
}

func (openvpn *openvpnProcess) Stop() error {
	waiter := sync.WaitGroup{}
	//TODO which to signal for close first ?
	//if we stop process before management, managemnt won't have a chance to send any commands from middlewares on stop
	//if we stop management first - it will miss important EXITING state from process
	waiter.Add(1)
	go func() {
		defer waiter.Done()
		openvpn.cmd.Stop()
	}()

	waiter.Add(1)
	go func() {
		defer waiter.Done()
		openvpn.management.Stop()
	}()

	waiter.Wait()
	return nil
}
