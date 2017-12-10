package command_run

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_dialog"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"os"
)

func NewCommand(vpnMiddlewares ...openvpn.ManagementMiddleware) *CommandRun {
	return &CommandRun{
		Output:      os.Stdout,
		OutputError: os.Stderr,

		IpifyClient:     ipify.NewClient(),
		MysteriumClient: server.NewClient(),
		NatService:      nat.NewService(),
		CommunicationServerFactory: func(identity dto_discovery.Identity) (communication.Server, dto_discovery.Contact) {
			address := nats_discovery.NewAddressForIdentity(identity)
			return nats_dialog.NewServer(address), address.GetContact()
		},
		SessionManager: &session.Manager{
			Generator: &session.Generator{},
		},
		vpnMiddlewares: vpnMiddlewares,
	}
}
