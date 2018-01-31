package monitor

import (
	"github.com/mysterium/node/cmd/monitor/node_provider"
	"github.com/mysterium/node/ip"
	"path/filepath"
	"time"
)

// NewCommand function creates new monitor command by given options
func NewCommand() *Command {
	return &Command{
		ipResolver: ip.NewResolverWithTimeout(5 * time.Second),
	}
}

// NewNodeProvider creates provider to return monitored nodes
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
		filepath.Join(options.DirectoryRuntime, "remember.status"),
	)
	return
}
