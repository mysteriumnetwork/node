package command_run

import (
	"errors"
	"fmt"
	"github.com/mysterium/node/ipify"
	"io"
	"os"
	"sync"

	log "github.com/cihub/seelog"
	command_client "github.com/mysterium/node/cmd/mysterium_client/command_run"
)

const MYSTERIUM_MONITOR_LOG_PREFIX = "[Mysterium.monitor] "

type commandRun struct {
	output      io.Writer
	outputError io.Writer

	ipifyClient ipify.Client
	waiter      sync.WaitGroup
}

func (cmd *commandRun) Run(options CommandOptions) error {
	nodeKeys := []string{"mysterium-vpn3"}

	ipOriginal, err := cmd.ipifyClient.GetIp()
	if err != nil {
		return errors.New("Failed to get original IP: " + err.Error())
	}

	for _, nodeKey := range nodeKeys {
		err := runVPNClient(&cmd.waiter, command_client.CommandOptions{
			NodeKey:          nodeKey,
			DirectoryRuntime: options.DirectoryRuntime,
		})
		if err != nil {
			return errors.New("Client starting error: " + err.Error())
		}

		ipForwarded, err := cmd.ipifyClient.GetIp()
		if err != nil {
			log.Warn(MYSTERIUM_MONITOR_LOG_PREFIX, "Forwarded IP not detected: ", err)
			continue
		}
		if ipForwarded == ipOriginal {
			log.Warn(MYSTERIUM_MONITOR_LOG_PREFIX, "Forwarded IP is the same")
			continue
		}
	}

	return nil
}

func (cmd *commandRun) Wait() error {
	cmd.waiter.Wait()
	return nil
}

func (cmd *commandRun) Kill() {

}

func runVPNClient(waiter *sync.WaitGroup, options command_client.CommandOptions) error {
	clientCommand := command_client.NewCommand()
	err := clientCommand.Run(options)
	if err != nil {
		return err
	}

	waiter.Add(1)
	go func() {
		defer waiter.Done()

		if err := clientCommand.Wait(); err != nil {
			fmt.Fprintln(os.Stderr, "Client stopped with error: ", err)
			os.Exit(1)
		}
	}()

	return nil
}
