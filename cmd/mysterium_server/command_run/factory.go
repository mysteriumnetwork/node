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
    "github.com/mysterium/node/identity"
)

func NewCommand(vpnMiddlewares ...openvpn.ManagementMiddleware) *CommandRun {
	return &CommandRun{
		Output:      os.Stdout,
		OutputError: os.Stderr,

		IpifyClient:     ipify.NewClient(),
		MysteriumClient: server.NewClient(),
		NatService:      nat.NewService(),
		DialogWaiterFactory: func(identity identity.Identity) (communication.DialogWaiter, dto_discovery.Contact) {
			address, err  := nats_discovery.NewAddressForIdentity(identity)
			if err != nil {
				panic(err)
			}
			return nats_dialog.NewDialogWaiter(address), address.GetContact()
		},
		SessionManager: &session.Manager{
			Generator: &session.Generator{},
		},
		vpnMiddlewares: vpnMiddlewares,
	}
}
