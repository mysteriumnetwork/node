package command_run

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_dialog"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

func NewCommand(options CommandOptions) *CommandRun {
	return NewCommandWith(
		options,
		server.NewClient(),
		ipify.NewClient(),
		nat.NewService(),
	)
}

func NewCommandWith(
	options CommandOptions,
	mysteriumClient server.Client,
	ipifyClient ipify.Client,
	natService nat.NATService,
) *CommandRun {

	ks := keystore.NewKeyStore(options.DirectoryKeystore, keystore.StandardScryptN, keystore.StandardScryptP)
	identityHandler := NewNodeIdentityHandler(
		identity.NewIdentityManager(ks),
		mysteriumClient,
		options.DirectoryKeystore,
	)

	return &CommandRun{
		identitySelector: func() (identity.Identity, error) {
			return identityHandler.Select(options.NodeKey)
		},
		ipifyClient:     ipifyClient,
		mysteriumClient: mysteriumClient,
		natService:      natService,
		dialogWaiterFactory: func(identity identity.Identity) (communication.DialogWaiter, dto_discovery.Contact) {
			address := nats_discovery.NewAddressForIdentity(identity)
			return nats_dialog.NewDialogWaiter(address), address.GetContact()
		},
		vpnServerFactory: func() *openvpn.Server {
			vpnServerConfig := openvpn.NewServerConfig(
				"10.8.0.0", "255.255.255.0",
				options.DirectoryConfig+"/ca.crt",
				options.DirectoryConfig+"/server.crt",
				options.DirectoryConfig+"/server.key",
				options.DirectoryConfig+"/dh.pem",
				options.DirectoryConfig+"/crl.pem",
				options.DirectoryConfig+"/ta.key",
			)
			return openvpn.NewServer(vpnServerConfig, options.DirectoryRuntime)
		},
	}
}
