package command_run

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_dialog"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

func NewCommand(vpnMiddlewares ...openvpn.ManagementMiddleware) *CommandRun {
	nats_discovery.Bootstrap()
	openvpn.Bootstrap()

	return &CommandRun{

		MysteriumClient: server.NewClient(),
		DialogEstablisherFactory: func(identity dto_discovery.Identity) communication.DialogEstablisher {
			return nats_dialog.NewDialogEstablisher(identity)
		},

		vpnMiddlewares: vpnMiddlewares,
	}
}
