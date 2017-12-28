package command_run

import (
	"github.com/mysterium/node/cmd/mysterium_monitor/command_run/node_provider"
	"github.com/mysterium/node/ipify"
	"time"
)

func NewCommand() *CommandRun {
	return &CommandRun{
		IpifyClient: ipify.NewClientWithTimeout(5 * time.Second),
	}
}

func NewNodeProvider(options CommandOptions) (nodeProvider node_provider.NodeProvider, err error) {
	if options.Node != "" {
		nodeProvider = node_provider.NewArrayProvider([]string{options.Node})
	} else {
		nodeProvider, err = node_provider.NewFileProvider(options.NodeFile)
		if err != nil {
			return
		}
	}

	nodeProvider = node_provider.NewRememberProvider(
		nodeProvider,
		options.DirectoryRuntime+"/remember.status",
	)
	return
}
