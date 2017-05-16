package command_run

import (
	"os"
)

func NewCommand() *commandRun {
	return &commandRun{
		output:      os.Stdout,
		outputError: os.Stderr,
	}
}
