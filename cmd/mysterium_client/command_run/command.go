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

	connectionManager := client_connection.NewManager(mysteriumClient, dialogEstablisherFactory, vpnClientFactory)

	router := tequilapi.NewApiRouter()
	endpoints.RegisterIdentitiesEndpoint(router, identityManager)
	endpoints.RegisterConnectionEndpoint(router, connectionManager)

	httpApiServer, err := tequilapi.NewServer(options.TequilaApiAddress, options.TequilaApiPort, router)
	if err != nil {
		return nil, err
	}

	return &CommandRun{
		mysteriumClient,
		connectionManager,
		httpApiServer,
	}, nil
}

func (cmd *CommandRun) Run() {
	cmd.httpApiServer.StartServing()
}

func (cmd *CommandRun) Wait() error {
	return cmd.httpApiServer.Wait()
}

func (cmd *CommandRun) Kill() {
	cmd.httpApiServer.Stop()
	cmd.connectionManager.Disconnect()
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
