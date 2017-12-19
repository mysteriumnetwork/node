package command_run

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysterium/node/bytescount_client"
	"github.com/mysterium/node/client_connection"
	"github.com/mysterium/node/communication"
    "github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	vpn_session "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/tequilapi"
	"github.com/mysterium/node/tequilapi/endpoints"
	"time"
)

type CommandRun struct {
	MysteriumClient server.Client

	DialogEstablisherFactory func(identity identity.Identity) communication.DialogEstablisher
	dialog                   communication.Dialog

	vpnMiddlewares []openvpn.ManagementMiddleware
	vpnClient      *openvpn.Client

	httpApiServer tequilapi.ApiServer
}

func (cmd *CommandRun) Run(options CommandOptions) (err error) {
	consumerId := identity.FromAddress("consumer1")

	session, err := cmd.MysteriumClient.SessionCreate(options.NodeKey)
	if err != nil {
		return err
	}
	proposal := session.ServiceProposal

	dialogEstablisher := cmd.DialogEstablisherFactory(consumerId)
	cmd.dialog, err = dialogEstablisher.CreateDialog(proposal.ProviderContacts[0])
	if err != nil {
		return err
	}

	vpnSession, err := vpn_session.RequestSessionCreate(cmd.dialog, proposal.Id)
	if err != nil {
		return err
	}

	vpnConfig, err := openvpn.NewClientConfigFromString(
		vpnSession.Config,
		options.DirectoryRuntime+"/client.ovpn",
	)
	if err != nil {
		return err
	}

	vpnMiddlewares := append(
		cmd.vpnMiddlewares,
		bytescount_client.NewMiddleware(cmd.MysteriumClient, session.Id, 1*time.Minute),
	)
	cmd.vpnClient = openvpn.NewClient(
		vpnConfig,
		options.DirectoryRuntime,
		vpnMiddlewares...,
	)
	if err := cmd.vpnClient.Start(); err != nil {
		return err
	}

	router := tequilapi.NewApiRouter()

	keystoreInstance := keystore.NewKeyStore(options.DirectoryKeystore, keystore.StandardScryptN, keystore.StandardScryptP)
	idm := identity.NewIdentityManager(keystoreInstance)
	mystClient := server.NewClient()
	endpoints.RegisterIdentitiesEndpoint(router, idm, mystClient)

	endpoints.RegisterConnectionEndpoint(router, client_connection.NewManager())

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
	cmd.dialog.Close()
	cmd.vpnClient.Stop()
	cmd.httpApiServer.Stop()
}
