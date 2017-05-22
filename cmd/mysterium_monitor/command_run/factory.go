package command_run

import (
	"github.com/mysterium/node/ipify"
	"os"
)

func NewCommand() *commandRun {
	return &commandRun{
		output:      os.Stdout,
		outputError: os.Stderr,

		ipifyClient: ipify.NewClient(),
	}
}
