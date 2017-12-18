package command_run

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysterium/node/client_connection"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/tequilapi"
	"github.com/mysterium/node/tequilapi/endpoints"
)

type CommandRun struct {
	MysteriumClient server.Client

	dialogEstablisherFactory client_connection.DialogEstablisherFactory

	httpApiServer tequilapi.ApiServer
}

func (cmd *CommandRun) Run(options CommandOptions) (err error) {

	router := tequilapi.NewApiRouter()

	keystoreInstance := keystore.NewKeyStore(options.DirectoryKeystore, keystore.StandardScryptN, keystore.StandardScryptP)
	idm := identity.NewIdentityManager(keystoreInstance)
	endpoints.RegisterIdentitiesEndpoint(router, idm)

	vpnManager := client_connection.NewVpnManager(cmd.MysteriumClient, cmd.dialogEstablisherFactory, options.DirectoryRuntime)
	endpoints.RegisterConnectionEndpoint(router, vpnManager)

	cmd.httpApiServer, err = tequilapi.StartNewServer(options.TequilaApiAddress, options.TequilaApiPort, router)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *CommandRun) Wait() error {
	return cmd.httpApiServer.Wait()
}

func (cmd *CommandRun) Kill() {
	cmd.httpApiServer.Stop()
}
