package command_run

import (
	"github.com/mysterium/node/cmd/mysterium_monitor/command_run/node_provider"
	"github.com/mysterium/node/ipify"
	"os"
)

func NewCommand() Command {
	return &commandRun{
		output:      os.Stdout,
		outputError: os.Stderr,

		ipifyClient: ipify.NewClient(),
	}
}

func NewNodeProvider(options CommandOptions) (nodeProvider node_provider.NodeProvider, err error) {
	if options.Node != "" {
		nodeProvider = node_provider.NewArrayProvider([]string{options.Node})
	} else {
		nodeProvider, err = node_provider.NewFileProvider(options.NodeFile)
	}
	return
}
