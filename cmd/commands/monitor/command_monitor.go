package monitor

import (
	"errors"
	command_client "github.com/mysterium/node/cmd/commands/client"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/openvpn"
	"sync"
	"time"
)

// Command represent Mysterium monitor, client which checks connection to nodes
type Command struct {
	ipResolver ip.Resolver
	ipOriginal string

	clientCommand *command_client.Command
	ipCheckWaiter sync.WaitGroup
	resultWriter  *resultWriter
}

// Run starts monitor command - does not block
func (cmd *Command) Run(options CommandOptions) error {
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

	cmd.ipOriginal, err = cmd.ipResolver.GetOutboundIP()
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
		cmd.checkClientIPWhenDisconnected()
	})

	return nil
}

// This is meant to be registered as VpnClient middleware:
//   state.NewMiddleware(cmd.checkClientIPWhenConnected)
func (cmd *Command) checkClientIPWhenConnected(state openvpn.State) error {
	if state == openvpn.STATE_CONNECTED {
		ipForwarded, err := cmd.ipResolver.GetOutboundIP()
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

func (cmd *Command) checkClientHandleTimeout() {
	<-time.After(10 * time.Second)

	cmd.resultWriter.NodeStatus("Client not connected")
	cmd.ipCheckWaiter.Done()
}

func (cmd *Command) checkClientIPWhenDisconnected() {
	ipForwarded, err := cmd.ipResolver.GetOutboundIP()
	if err != nil {
		cmd.resultWriter.NodeError("Disconnect IP not detected", err)
		return
	}

	if ipForwarded != cmd.ipOriginal {
		cmd.resultWriter.NodeStatus("Disconnect IP does not match original")
		return
	}
}

// Wait blocks until monitoring is finished
func (cmd *Command) Wait() error {
	return nil
}

// Kill stops monitoring client
func (cmd *Command) Kill() {
	cmd.clientCommand.Kill()
}
