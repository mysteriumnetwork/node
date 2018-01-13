package command_run

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysterium/node/client_connection"
	"github.com/mysterium/node/communication"
	nats_dialog "github.com/mysterium/node/communication/nats/dialog"
	nats_discovery "github.com/mysterium/node/communication/nats/discovery"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/tequilapi"
	"github.com/mysterium/node/tequilapi/endpoints"
)

//NewCommand function created new client command with options passed from commandline
func NewCommand(options CommandOptions) *CommandRun {
	return NewCommandWith(
		options,
		server.NewClient(),
	)
}

//NewCommandWith does the same as NewCommand with posibility to override mysterium api client for external communication
func NewCommandWith(
	options CommandOptions,
	mysteriumClient server.Client,
) *CommandRun {
	nats_discovery.Bootstrap()
	openvpn.Bootstrap()

	keystoreInstance := keystore.NewKeyStore(options.DirectoryKeystore, keystore.StandardScryptN, keystore.StandardScryptP)

	identityManager := identity.NewIdentityManager(keystoreInstance)

	dialogEstablisherFactory := func(myIdentity identity.Identity) communication.DialogEstablisher {
		return nats_dialog.NewDialogEstablisher(myIdentity, identity.NewSigner(keystoreInstance, myIdentity))
	}

	signerFactory := func(id identity.Identity) identity.Signer {
		return identity.NewSigner(keystoreInstance, id)
	}

	vpnClientFactory := client_connection.ConfigureVpnClientFactory(mysteriumClient, options.DirectoryRuntime, signerFactory)

	connectionManager := client_connection.NewManager(mysteriumClient, dialogEstablisherFactory, vpnClientFactory)

	router := tequilapi.NewApiRouter()
	endpoints.AddRoutesForIdentities(router, identityManager, mysteriumClient, signerFactory)
	endpoints.AddRoutesForConnection(router, connectionManager)
	endpoints.AddRoutesForProposals(router, mysteriumClient)

	httpApiServer := tequilapi.NewServer(options.TequilaApiAddress, options.TequilaApiPort, router)

	return &CommandRun{
		connectionManager,
		httpApiServer,
	}
}

//CommandRun represent entry point for MysteriumVpn client with top level components
type CommandRun struct {
	connectionManager client_connection.Manager
	httpApiServer     tequilapi.ApiServer
}

//Run starts tequilaApi service - does not block
func (cmd *CommandRun) Run() error {
	err := cmd.httpApiServer.StartServing()
	if err != nil {
		return err
	}
	port, err := cmd.httpApiServer.Port()
	if err != nil {
		return err
	}

	fmt.Printf("Api started on: %d\n", port)
	return nil
}

//Wait blocks until tequilapi service is stopped
func (cmd *CommandRun) Wait() error {
	return cmd.httpApiServer.Wait()
}

//Kill stops tequilapi service
func (cmd *CommandRun) Kill() {
	cmd.httpApiServer.Stop()
	cmd.connectionManager.Disconnect()
}
