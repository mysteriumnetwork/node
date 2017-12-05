package command_run

import (
	"github.com/mysterium/node/bytescount_client"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/openvpn"
	vpn_session "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/tequilapi"
	"io"
	"time"
)

type CommandRun struct {
	Output      io.Writer
	OutputError io.Writer

	MysteriumClient server.Client

	CommunicationClientFactory func(identity dto_discovery.Identity) communication.Client
	communicationClient        communication.Client

	vpnMiddlewares []openvpn.ManagementMiddleware
	vpnClient      *openvpn.Client

	httpApiServer tequilapi.ApiServer
}

func (cmd *CommandRun) Run(options CommandOptions) (err error) {
	consumerId := dto_discovery.Identity("consumer1")

	session, err := cmd.MysteriumClient.SessionCreate(options.NodeKey)
	if err != nil {
		return err
	}

	cmd.communicationClient = cmd.CommunicationClientFactory(consumerId)
	proposal := session.ServiceProposal
	sender, _, err := cmd.communicationClient.CreateDialog(proposal.ProviderContacts[0].Definition)
	if err != nil {
		return err
	}

	vpnSession, err := vpn_session.RequestSessionCreate(sender, proposal.Id)
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

	apiEndpoints := tequilapi.NewApiEndpoints()
	//TODO additional endpoint registration can go here i.e apiEndpoints.GET("/path", httprouter.Handle function)

	cmd.httpApiServer, err = tequilapi.StartNewServer(options.TequilaApiAddress, options.TequilaApiPort, apiEndpoints)
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
	cmd.communicationClient.Stop()
	cmd.vpnClient.Stop()
}
