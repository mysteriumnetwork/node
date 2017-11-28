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

const NODE_KEY = "fake"
const NODE_IP = "127.0.0.1"
const NODE_DIRECTORY_CONFIG = "bin/tls"
const CLIENT_DIRECTORY_RUNTIME = "build/fake"

func main() {
	waiter := &sync.WaitGroup{}
	mysteriumClient := server.NewClientFake()

	serverCommand := command_server.NewCommand()
	serverCommand.Output = os.Stdout
	serverCommand.OutputError = os.Stderr
	serverCommand.IpifyClient = ipify.NewClientFake(NODE_IP)
	serverCommand.MysteriumClient = mysteriumClient
	serverCommand.NatService = nat.NewServiceFake()
	runServer(serverCommand, waiter)

	clientCommand := command_client.NewCommand()
	clientCommand.Output = os.Stdout
	clientCommand.OutputError = os.Stderr
	clientCommand.MysteriumClient = mysteriumClient
	runClient(clientCommand, waiter)

	waiter.Wait()
}

func runServer(serverCommand *command_server.CommandRun, waiter *sync.WaitGroup) {
	err := serverCommand.Run(command_server.CommandOptions{
		NodeKey:          NODE_KEY,
		DirectoryConfig:  NODE_DIRECTORY_CONFIG,
		DirectoryRuntime: CLIENT_DIRECTORY_RUNTIME,
	})
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
	err := clientCommand.Run(command_client.CommandOptions{
		NodeKey:          NODE_KEY,
		DirectoryRuntime: CLIENT_DIRECTORY_RUNTIME,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Client starting error: ", err)
		os.Exit(1)
	}

	waiter.Add(1)
	go func() {
		defer waiter.Done()

		if err = clientCommand.Wait(); err != nil {
			fmt.Fprintln(os.Stderr, "Client stopped with error: ", err)
			os.Exit(1)
		}
	}()
}
