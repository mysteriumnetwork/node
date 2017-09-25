package command_run

import (
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	"os"
)

func NewCommand(vpnMiddlewares ...openvpn.ManagementMiddleware) *CommandRun {
	return &CommandRun{
		Output:      os.Stdout,
		OutputError: os.Stderr,

		IpifyClient:         ipify.NewClient(),
		MysteriumClient:     server.NewClient(),
		NatService:          nat.NewService(),
		CommunicationServer: nats.NewServer(),
		vpnMiddlewares:      vpnMiddlewares,
	}
}
