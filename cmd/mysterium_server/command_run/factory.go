package command_run

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	"io"
	"os"
)

func NewCommand() Command {
	return NewCommandWithDependencies(
		os.Stdout,
		os.Stderr,
		ipify.NewClient(),
		server.NewClient(),
		nat.NewService(),
		nats.NewChannel(),
	)
}

func NewCommandWithDependencies(
	output io.Writer,
	outputError io.Writer,
	ipifyClient ipify.Client,
	mysteriumClient server.Client,
	natService nat.NATService,
	communicationChannel communication.Channel,
) Command {
	return &commandRun{
		output:      output,
		outputError: outputError,

		ipifyClient:          ipifyClient,
		mysteriumClient:      mysteriumClient,
		natService:           natService,
		communicationChannel: communicationChannel,
		vpnMiddlewares:       make([]openvpn.ManagementMiddleware, 0),
	}
}
