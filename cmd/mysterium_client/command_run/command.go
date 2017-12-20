package command_run

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysterium/node/bytescount_client"
	"github.com/mysterium/node/client_connection"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_dialog"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/server/dto"
	"github.com/mysterium/node/tequilapi"
	"github.com/mysterium/node/tequilapi/endpoints"
	"path/filepath"
	"time"
)

type CommandRun struct {
	//TODO this must disappear or become a private field
	MysteriumClient server.Client

	identityManager identity.IdentityManagerInterface

	connectionManager client_connection.Manager

	httpApiServer tequilapi.ApiServer
}

func NewCommand(options CommandOptions) (*CommandRun, error) {
	nats_discovery.Bootstrap()
	openvpn.Bootstrap()

	mysteriumClient := server.NewClient()

	keystoreInstance := keystore.NewKeyStore(options.DirectoryKeystore, keystore.StandardScryptN, keystore.StandardScryptP)

	identityManager := identity.NewIdentityManager(keystoreInstance)

	dialogEstablisherFactory := func(identity identity.Identity) communication.DialogEstablisher {
		return nats_dialog.NewDialogEstablisher(identity)
	}

	vpnClientFactory := configureVpnClientFactory(mysteriumClient, options.DirectoryRuntime)

	vpnManager := client_connection.NewManager(mysteriumClient, dialogEstablisherFactory, vpnClientFactory)

	httpApiServer, err := tequilapi.NewServer(options.TequilaApiAddress, options.TequilaApiPort)
	if err != nil {
		return nil, err
	}

	return &CommandRun{
		mysteriumClient,
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

func configureVpnClientFactory(mysteriumApiClient server.Client, vpnClientRuntimeDirectory string) client_connection.VpnClientFactory {
	return func(vpnSession *session.VpnSession, session *dto.Session) (openvpn.Client, error) {
		vpnConfig, err := openvpn.NewClientConfigFromString(
			vpnSession.Config,
			filepath.Join(vpnClientRuntimeDirectory, "client.ovpn"),
		)
		if err != nil {
			return nil, err
		}

		vpnMiddlewares := []openvpn.ManagementMiddleware{
			bytescount_client.NewMiddleware(mysteriumApiClient, session.Id, 1*time.Minute),
		}
		return openvpn.NewClient(
			vpnConfig,
			vpnClientRuntimeDirectory,
			vpnMiddlewares...,
		), nil

	}
}
