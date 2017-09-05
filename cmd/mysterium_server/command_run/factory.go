package command_run

import (
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	"github.com/nats-io/go-nats"
	"io"
	"os"
)

func NewCommand() Command {
	communicationOptions := nats.GetDefaultOptions()
	communicationOptions.Servers = []string{
		"nats://127.0.0.1:4222",
	}

	return &commandRun{
		output:      os.Stdout,
		outputError: os.Stderr,

		ipifyClient:          ipify.NewClient(),
		mysteriumClient:      server.NewClient(),
		communicationOptions: communicationOptions,
		vpnMiddlewares:       make([]openvpn.ManagementMiddleware, 0),

		natService: nat.NewService(),
	}
}

func NewCommandWithDependencies(
	output io.Writer,
	outputError io.Writer,
	ipifyClient ipify.Client,
	mysteriumClient server.Client,
	vpnMiddlewares ...openvpn.ManagementMiddleware,
) Command {
	return &commandRun{
		output:      output,
		outputError: outputError,

		ipifyClient:     ipifyClient,
		mysteriumClient: mysteriumClient,
		vpnMiddlewares:  vpnMiddlewares,
	}
}
