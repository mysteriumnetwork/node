package command_run

import (
	"github.com/MysteriumNetwork/node/ipify"
	"github.com/MysteriumNetwork/node/nat"
	"github.com/MysteriumNetwork/node/server"
	"io"
	"os"
)

func NewCommand() *commandRun {
	return &commandRun{
		output:          os.Stdout,
		outputError:     os.Stderr,
		ipifyClient:     ipify.NewClient(),
		mysteriumClient: server.NewClient(),
		natService:      nat.NewService(),
	}
}

func NewCommandWithDependencies(
	output io.Writer,
	outputError io.Writer,
	ipifyClient ipify.Client,
	mysteriumClient server.Client,
) *commandRun {
	return &commandRun{
		output:          output,
		outputError:     outputError,
		ipifyClient:     ipifyClient,
		mysteriumClient: mysteriumClient,
	}
}
