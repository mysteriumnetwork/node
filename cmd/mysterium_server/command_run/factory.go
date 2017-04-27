package command_run

import "os"

func NewCommandRun() *commandRun {
	return &commandRun{
		output:      os.Stdout,
		outputError: os.Stderr,
	}
}
