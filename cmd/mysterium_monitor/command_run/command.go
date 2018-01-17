package command_run

import (
	"errors"
	command_client "github.com/mysterium/node/cmd/mysterium_client/run"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/openvpn/middlewares/client/state"
	"sync"
	"time"
)

type CommandRun struct {
	IpifyClient ipify.Client
	ipOriginal  string

	clientCommand *command_client.CommandRun
	ipCheckWaiter sync.WaitGroup
	resultWriter  *resultWriter
}

func (cmd *CommandRun) Run(options CommandOptions) error {
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

	cmd.ipOriginal, err = cmd.IpifyClient.GetOutboundIP()
	if err != nil {
		return errors.New("Failed to get original IP: " + err.Error())
	}

	cmd.clientCommand = command_client.NewCommand(command_client.CommandOptions{
		DirectoryRuntime: options.DirectoryRuntime,
	})

	nodeProvider.WithEachNode(func(nodeKey string) {
		cmd.resultWriter.NodeStart(nodeKey)
		cmd.ipCheckWaiter.Add(1)

		//TODO here we need to make Tequilapi call with connect to node by key
		err = cmd.clientCommand.Run()
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

// This is ment to be registered as VpnClient middleware:
//   state.NewMiddleware(cmd.checkClientIpWhenConnected)
func (cmd *CommandRun) checkClientIpWhenConnected(state state.State) error {
	if state == state.STATE_CONNECTED {
		ipForwarded, err := cmd.IpifyClient.GetOutboundIP()
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

func (cmd *CommandRun) checkClientHandleTimeout() {
	<-time.After(10 * time.Second)

	cmd.resultWriter.NodeStatus("Client not connected")
	cmd.ipCheckWaiter.Done()
}

func (cmd *CommandRun) checkClientIpWhenDisconnected() {
	ipForwarded, err := cmd.IpifyClient.GetOutboundIP()
	if err != nil {
		cmd.resultWriter.NodeError("Disconnect IP not detected", err)
		return
	}

	if ipForwarded != cmd.ipOriginal {
		cmd.resultWriter.NodeStatus("Disconnect IP does not match original")
		return
	}
}

func (cmd *CommandRun) Wait() error {
	return nil
}

func (cmd *CommandRun) Kill() {
	cmd.clientCommand.Kill()
}
