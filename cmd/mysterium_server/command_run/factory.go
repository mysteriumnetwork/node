package command_run

import (
	"os"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/server"
	"io"
)

func NewCommand() *commandRun {
	return &commandRun{
		output:      os.Stdout,
		outputError: os.Stderr,
		ipifyClient: ipify.NewClient(),
		mysteriumClient: server.NewClient(),
	}
}

func NewCommandWithDependencies(
	output io.Writer,
	outputError io.Writer,
	ipifyClient ipify.Client,
	mysteriumClient server.Client,
) *commandRun {
	return &commandRun{
		output: output,
		outputError: outputError,
		ipifyClient: ipifyClient,
		mysteriumClient: mysteriumClient,
	}
}
