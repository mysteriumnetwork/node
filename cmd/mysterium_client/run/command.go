package run

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
	tequilapi_endpoints "github.com/mysterium/node/tequilapi/endpoints"
	"os"
	"os/signal"
	"syscall"
)

//NewCommand function created new client command with options passed from commandline
func NewCommand(options CommandOptions) *CommandRun {
	return NewCommandWith(
		options,
		server.NewClient(),
	)
}

//NewCommandWith does the same as NewCommand with possibility to override mysterium api client for external communication
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
	tequilapi_endpoints.AddRoutesForIdentities(router, identityManager, mysteriumClient, signerFactory)
	tequilapi_endpoints.AddRoutesForConnection(router, connectionManager)
	tequilapi_endpoints.AddRoutesForProposals(router, mysteriumClient)

	httpApiServer := tequilapi.NewServer(options.TequilapiAddress, options.TequilapiPort, router)
	cmd := &CommandRun{
		connectionManager,
		httpApiServer,
	}
	sigterm := make(chan os.Signal, 2)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go sarahConnor(sigterm, cmd)
	return cmd
}

// she handles Terminator signals
func sarahConnor(terminator chan os.Signal, cmd *CommandRun) {
	<-terminator
	err := cmd.Kill()
	if err != nil {
		fmt.Printf("Unable to disconnect %q\n", err.Error())
	}
	fmt.Println("Good bye")
	os.Exit(1)
}

//CommandRun represent entry point for MysteriumVpn client with top level components
type CommandRun struct {
	connectionManager client_connection.Manager
	httpApiServer     tequilapi.ApiServer
}

//Run starts Tequilapi service - does not block
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
func (cmd *CommandRun) Kill() error {
	err := cmd.connectionManager.Disconnect()
	if err != nil {
		return err
	}

	cmd.httpApiServer.Stop()
	fmt.Printf("Api stopped\n")

	return nil
}
