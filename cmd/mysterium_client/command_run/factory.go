package command_run

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	"io"
	"os"
)

func NewCommand() Command {
	return &commandRun{
		output:      os.Stdout,
		outputError: os.Stderr,

		mysteriumClient: server.NewClient(),
		vpnMiddlewares:  make([]openvpn.ManagementMiddleware, 0),
	}
}

func NewCommandWithDependencies(
	output io.Writer,
	outputError io.Writer,

	mysteriumClient server.Client,
	vpnMiddlewares ...openvpn.ManagementMiddleware,

) Command {
	return &commandRun{
		output:      output,
		outputError: outputError,

		mysteriumClient: mysteriumClient,
		vpnMiddlewares:  vpnMiddlewares,
	}
}
