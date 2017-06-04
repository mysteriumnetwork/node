package command_run

import (
	"errors"
	"github.com/mysterium/node/ipify"
	"io"

	log "github.com/cihub/seelog"
	command_client "github.com/mysterium/node/cmd/mysterium_client/command_run"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/state_client"
	"time"
)

const MYSTERIUM_MONITOR_LOG_PREFIX = "[Mysterium.monitor] "

type commandRun struct {
	output      io.Writer
	outputError io.Writer

	ipifyClient ipify.Client
	ipOriginal  string

	clientCommand command_client.Command
}

func (cmd *commandRun) Run(options CommandOptions) error {
	var err error
	nodeKeys := []string{"mysterium-vpn2", "mysterium-vpn3"}

	cmd.ipOriginal, err = cmd.ipifyClient.GetIp()
	if err != nil {
		return errors.New("Failed to get original IP: " + err.Error())
	}

	cmd.clientCommand = command_client.NewCommandWithDependencies(
		cmd.output,
		cmd.outputError,
		server.NewClient(),
		state_client.NewMiddleware(cmd.checkClientIpWhenConnected),
	)
	for _, nodeKey := range nodeKeys {
		err = cmd.clientCommand.Run(command_client.CommandOptions{
			NodeKey:          nodeKey,
			DirectoryRuntime: options.DirectoryRuntime,
		})
		if err != nil {
			log.Warn(MYSTERIUM_MONITOR_LOG_PREFIX, "Client not connected")
			return errors.New("Client starting error: " + err.Error())
		}

		<-time.After(2 * time.Second)
		log.Warn(MYSTERIUM_MONITOR_LOG_PREFIX, "Client not connected")

		cmd.clientCommand.Kill()
		cmd.checkClientIpWhenDisconnected()
	}

	return nil
}

func (cmd *commandRun) checkClientIpWhenConnected(state state_client.State) error {
	if state == state_client.STATE_CONNECTED {
		ipForwarded, err := cmd.ipifyClient.GetIp()
		if err != nil {
			log.Warn(MYSTERIUM_MONITOR_LOG_PREFIX, "Forwarded IP not detected: ", err)
			return nil
		}

		if ipForwarded == cmd.ipOriginal {
			log.Warn(MYSTERIUM_MONITOR_LOG_PREFIX, "Forwarded IP matches original")
			return nil
		}
	}
	return nil
}

func (cmd *commandRun) checkClientIpWhenDisconnected() {
	ipForwarded, err := cmd.ipifyClient.GetIp()
	if err != nil {
		log.Warn(MYSTERIUM_MONITOR_LOG_PREFIX, "Disconnect IP not detected: ", err)
		return
	}

	if ipForwarded != cmd.ipOriginal {
		log.Warn(MYSTERIUM_MONITOR_LOG_PREFIX, "Disconnect IP does not match original")
		return
	}
}

func (cmd *commandRun) Wait() error {
	return nil
}

func (cmd *commandRun) Kill() {
	cmd.clientCommand.Kill()
}
