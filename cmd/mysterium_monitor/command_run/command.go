package command_run

import (
	"fmt"
	"io"
	"os"
	"sync"

	command_client "github.com/mysterium/node/cmd/mysterium_client/command_run"
)

type commandRun struct {
	output      io.Writer
	outputError io.Writer

	waiter sync.WaitGroup
}

func (cmd *commandRun) Run(options CommandOptions) error {
	nodeKeys := []string{"mysterium-vpn3"}

	for _, nodeKey := range nodeKeys {
		runVPNClient(&cmd.waiter, command_client.CommandOptions{
			NodeKey:          nodeKey,
			DirectoryRuntime: options.DirectoryRuntime,
		})
	}

	return nil
}

func (cmd *commandRun) Wait() error {
	cmd.waiter.Wait()
	return nil
}

func (cmd *commandRun) Kill() {

}

func runVPNClient(waiter *sync.WaitGroup, options command_client.CommandOptions) {
	clientCommand := command_client.NewCommand()
	err := clientCommand.Run(options)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Client starting error: ", err)
		os.Exit(1)
	}

	waiter.Add(1)
	go func() {
		defer waiter.Done()

		if err := clientCommand.Wait(); err != nil {
			fmt.Fprintln(os.Stderr, "Client stopped with error: ", err)
			os.Exit(1)
		}
	}()
}
