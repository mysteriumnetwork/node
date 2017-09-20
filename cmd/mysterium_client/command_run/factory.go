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
		nats.NewClient(),
	)
}

func NewCommandWithDependencies(
	output io.Writer,
	outputError io.Writer,
	mysteriumClient server.Client,
	communicationClient communication.Client,
	vpnMiddlewares ...openvpn.ManagementMiddleware,
) Command {
	return &commandRun{
		output:      output,
		outputError: outputError,

		mysteriumClient:     mysteriumClient,
		communicationClient: communicationClient,

		vpnMiddlewares: vpnMiddlewares,
	}
}
