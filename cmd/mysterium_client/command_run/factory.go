package command_run

import (
	"os"
	"github.com/MysteriumNetwork/node/server"
	"io"
)

func NewCommand() *commandRun {
	return &commandRun{
		output: os.Stdout,
		outputError: os.Stderr,
		mysteriumClient: server.NewClient(),
	}
}

func NewCommandWithDependencies(
	output io.Writer,
	outputError io.Writer,
	mysteriumClient server.Client,
) *commandRun {
	return &commandRun{
		output: output,
		outputError: outputError,
		mysteriumClient: mysteriumClient,
	}
}
