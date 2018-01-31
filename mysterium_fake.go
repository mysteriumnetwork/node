package main

import (
	"fmt"
	"github.com/mysterium/node/cmd"
	command_client "github.com/mysterium/node/cmd/client"
	command_server "github.com/mysterium/node/cmd/server"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/server"
	"os"
	"sync"
)

const NodeIP = "127.0.0.1"
const NodeDirectoryConfig = "bin/tls"
const ClientDirectoryRuntime = "build/fake"

func main() {
	waiter := &sync.WaitGroup{}
	mysteriumClient := server.NewClientFake()

	serverCommand := command_server.NewCommandWith(
		command_server.CommandOptions{
			DirectoryConfig:  NodeDirectoryConfig,
			DirectoryRuntime: ClientDirectoryRuntime,
		},
		mysteriumClient,
		ip.NewFakeResolver(NodeIP),
		nat.NewServiceFake(),
	)
	cmd.NewTerminator(serverCommand)
	runServer(serverCommand, waiter)

	clientCommand := command_client.NewCommandWith(
		command_client.CommandOptions{
			DirectoryRuntime: ClientDirectoryRuntime,
		},
		mysteriumClient,
	)
	cmd.NewTerminator(clientCommand)
	runClient(clientCommand, waiter)

	waiter.Wait()
}

func runServer(serverCommand *command_server.Command, waiter *sync.WaitGroup) {
	err := serverCommand.Run()
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

func runClient(clientCommand *command_client.Command, waiter *sync.WaitGroup) {
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
