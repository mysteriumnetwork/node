package command_run

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"os"
)

func NewCommand(vpnMiddlewares ...openvpn.ManagementMiddleware) *CommandRun {
	ipifyClient := ipify.NewClient()

	return &CommandRun{
		Output:      os.Stdout,
		OutputError: os.Stderr,

		IpifyClient:     ipifyClient,
		MysteriumClient: server.NewClient(),
		NatService:      nat.NewService(),
		CommunicationServerFactory: func(identity dto_discovery.Identity) communication.Server {
			return nats.NewServer(identity, ipifyClient)
		},
		vpnMiddlewares: vpnMiddlewares,
	}
}
