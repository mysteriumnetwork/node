package command_run

import (
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	"os"
)

func NewCommand(vpnMiddlewares ...openvpn.ManagementMiddleware) *CommandRun {
	return &CommandRun{
		Output:      os.Stdout,
		OutputError: os.Stderr,

		MysteriumClient:     server.NewClient(),
		CommunicationClient: nats.NewClient(),

		vpnMiddlewares: vpnMiddlewares,
	}
}
