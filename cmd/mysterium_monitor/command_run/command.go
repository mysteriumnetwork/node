package command_run

import (
	"errors"
	"github.com/mysterium/node/ipify"
	"io"

	command_client "github.com/mysterium/node/cmd/mysterium_client/command_run"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/state_client"
	"sync"
	"time"
)

type commandRun struct {
	output      io.Writer
	outputError io.Writer

	ipifyClient ipify.Client
	ipOriginal  string

	clientCommand command_client.Command
	ipCheckWaiter sync.WaitGroup
	resultWriter  *resultWriter
}

func (cmd *commandRun) Run(options CommandOptions) error {
	var err error

	cmd.resultWriter, err = NewResultWriter(options.ResultFile)
	if err != nil {
		return err
	}
	defer cmd.resultWriter.Close()

	nodeProvider, err := NewNodeProvider(options)
	if err != nil {
		return err
	}
	defer nodeProvider.Close()

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

	nodeProvider.WithEachNode(func(nodeKey string) {
		cmd.resultWriter.NodeStart(nodeKey)
		cmd.ipCheckWaiter.Add(1)

		err = cmd.clientCommand.Run(command_client.CommandOptions{
			NodeKey:          nodeKey,
			DirectoryRuntime: options.DirectoryRuntime,
		})
		if err != nil {
			cmd.resultWriter.NodeError("Client starting error", err)
			return
		}

		go cmd.checkClientHandleTimeout()

		cmd.ipCheckWaiter.Wait()
		cmd.clientCommand.Kill()
		cmd.checkClientIpWhenDisconnected()
	})

	return nil
}

func (cmd *commandRun) checkClientIpWhenConnected(state state_client.State) error {
	if state == state_client.STATE_CONNECTED {
		ipForwarded, err := cmd.ipifyClient.GetIp()
		if err != nil {
			cmd.resultWriter.NodeError("Forwarded IP not detected", err)
			cmd.ipCheckWaiter.Done()
			return nil
		}

		if ipForwarded == cmd.ipOriginal {
			cmd.resultWriter.NodeStatus("Forwarded IP matches original")
			cmd.ipCheckWaiter.Done()
			return nil
		}

		cmd.resultWriter.NodeStatus("OK")
		cmd.ipCheckWaiter.Done()
	}
	return nil
}

func (cmd *commandRun) checkClientHandleTimeout() {
	<-time.After(10 * time.Second)

	cmd.resultWriter.NodeStatus("Client not connected")
	cmd.ipCheckWaiter.Done()
}

func (cmd *commandRun) checkClientIpWhenDisconnected() {
	ipForwarded, err := cmd.ipifyClient.GetIp()
	if err != nil {
		cmd.resultWriter.NodeError("Disconnect IP not detected", err)
		return
	}

	if ipForwarded != cmd.ipOriginal {
		cmd.resultWriter.NodeStatus("Disconnect IP does not match original")
		return
	}
}

func (cmd *commandRun) Wait() error {
	return nil
}

func (cmd *commandRun) Kill() {
	cmd.clientCommand.Kill()
}
