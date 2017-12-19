package command_run

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysterium/node/client_connection"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_dialog"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/tequilapi"
	"github.com/mysterium/node/tequilapi/endpoints"
)

type CommandRun struct {
	MysteriumClient server.Client

	dialogEstablisherFactory client_connection.DialogEstablisherFactory

	//find interface in ethereum?
	keystore *keystore.KeyStore

	identityManager identity.IdentityManagerInterface

	connectionManager client_connection.Manager

	httpApiServer tequilapi.ApiServer
}

func NewCommand(options CommandOptions) (*CommandRun, error) {
	nats_discovery.Bootstrap()
	openvpn.Bootstrap()

	mysteriumClient := server.NewClient()

	dialogEstablisherFactory := func(identity dto.Identity) communication.DialogEstablisher {
		return nats_dialog.NewDialogEstablisher(identity)
	}

	keystoreInstance := keystore.NewKeyStore(options.DirectoryKeystore, keystore.StandardScryptN, keystore.StandardScryptP)

	identityManager := identity.NewIdentityManager(keystoreInstance)

	vpnManager := client_connection.NewVpnManager(mysteriumClient, dialogEstablisherFactory, options.DirectoryRuntime)

	httpApiServer, err := tequilapi.NewServer(options.TequilaApiAddress, options.TequilaApiPort)
	if err != nil {
		return nil, err
	}

	return &CommandRun{
		mysteriumClient,
		dialogEstablisherFactory,
		keystoreInstance,
		identityManager,
		vpnManager,
		httpApiServer,
	}, nil
}

func (cmd *CommandRun) Run() {
	router := tequilapi.NewApiRouter()
	endpoints.RegisterIdentitiesEndpoint(router, cmd.identityManager)
	endpoints.RegisterConnectionEndpoint(router, cmd.connectionManager)
	cmd.httpApiServer.StartServing(router)
}

func (cmd *CommandRun) Wait() error {
	return cmd.httpApiServer.Wait()
}

func (cmd *CommandRun) Kill() {
	cmd.httpApiServer.Stop()
}
