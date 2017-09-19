package command_run

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	"io"
	"os"
)

func NewCommand() Command {
	return NewCommandWithDependencies(
		os.Stdout,
		os.Stderr,
		server.NewClient(),
		nats.NewChannel(),
	)
}

func NewCommandWithDependencies(
	output io.Writer,
	outputError io.Writer,
	mysteriumClient server.Client,
	communicationChannel communication.Channel,
	vpnMiddlewares ...openvpn.ManagementMiddleware,
) Command {
	return &commandRun{
		output:      output,
		outputError: outputError,

		mysteriumClient:      mysteriumClient,
		communicationChannel: communicationChannel,

		vpnMiddlewares: vpnMiddlewares,
	}
}
