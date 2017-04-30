package main

import (
	"os"
	"fmt"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/server"
	command_server "github.com/mysterium/node/cmd/mysterium_server/command_run"
	command_client "github.com/mysterium/node/cmd/mysterium_client/command_run"
)

const NODE_KEY = "fake"
const NODE_IP = "127.0.0.1"
const NODE_DIRECTORY_CONFIG = "bin/tls"
const CLIENT_DIRECTORY_RUNTIME = "build/fake"

func main() {
	mysteriumClient := server.NewClientFake()
	ipifyClient := ipify.NewClientFake(NODE_IP)

	serverCommand := command_server.NewCommandWithDependencies(os.Stdout, os.Stderr, ipifyClient, mysteriumClient)
	err := serverCommand.Run(command_server.CommandOptions{
		NodeKey:         NODE_KEY,
		DirectoryConfig: NODE_DIRECTORY_CONFIG,
		DirectoryRuntime: CLIENT_DIRECTORY_RUNTIME,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Server starting error: ", err)
		os.Exit(1)
	}

	clientCommand := command_client.NewCommandWithDependencies(os.Stdout, os.Stderr, mysteriumClient)
	err = clientCommand.Run(command_client.CommandOptions{
		NodeKey:          NODE_KEY,
		DirectoryRuntime: CLIENT_DIRECTORY_RUNTIME,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Client starting error: ", err)
		os.Exit(1)
	}

	clientCommand.Wait()
}