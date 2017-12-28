package main

import (
	"fmt"
	command_client "github.com/mysterium/node/cmd/mysterium_client/command_run"
	command_server "github.com/mysterium/node/cmd/mysterium_server/command_run"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/server"
	"os"
	"sync"
)

const NodeIp = "127.0.0.1"
const NodeDirectoryConfig = "bin/tls"
const ClientDirectoryRuntime = "build/fake"

func main() {
	waiter := &sync.WaitGroup{}
	mysteriumClient := server.NewClientFake()

	serverCommand := command_server.NewCommand(command_server.CommandOptions{
		DirectoryConfig:  NodeDirectoryConfig,
		DirectoryRuntime: ClientDirectoryRuntime,
	})
	serverCommand.Output = os.Stdout
	serverCommand.OutputError = os.Stderr
	serverCommand.IpifyClient = ipify.NewClientFake(NodeIp)
	serverCommand.MysteriumClient = mysteriumClient
	serverCommand.NatService = nat.NewServiceFake()
	runServer(serverCommand, waiter)

	clientCommand := command_client.NewCommand(command_client.CommandOptions{
		DirectoryRuntime: ClientDirectoryRuntime,
	})

	//TODO refactor this internal variable override
	clientCommand.MysteriumClient = mysteriumClient
	runClient(clientCommand, waiter)

	waiter.Wait()
}

func runServer(serverCommand *command_server.CommandRun, waiter *sync.WaitGroup) {
	err := serverCommand.Run(command_server.CommandOptions{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Server starting error: ", err)
		os.Exit(1)
	}

	waiter.Add(1)
	go func() {
		defer waiter.Done()

		if err = serverCommand.Wait(); err != nil {
			fmt.Fprintln(os.Stderr, "Server stopped with error: ", err)
			os.Exit(1)
		}
	}()
}

func runClient(clientCommand *command_client.CommandRun, waiter *sync.WaitGroup) {
	err := clientCommand.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Client runtime error: ", err)
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
