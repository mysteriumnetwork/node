package command_run

import (
	"github.com/mysterium/node/bytescount_client"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"io"
	"time"
	"github.com/mysterium/node/openvpn/service_discovery/dto"
)

type CommandRun struct {
	Output      io.Writer
	OutputError io.Writer

	MysteriumClient server.Client

	CommunicationClientFactory func(identity dto_discovery.Identity) communication.Client
	communicationClient        communication.Client

	vpnMiddlewares []openvpn.ManagementMiddleware
	vpnClient      *openvpn.Client
}

func (cmd *CommandRun) Run(options CommandOptions) (err error) {
	consumerId := dto_discovery.Identity("consumer1")

	dto.Initialize()

	// rename this
	vpnSession, err := cmd.MysteriumClient.SessionCreate(options.NodeKey)
	if err != nil {
		return err
	}

	cmd.communicationClient = cmd.CommunicationClientFactory(consumerId)

	_, _, err = cmd.communicationClient.CreateDialog(vpnSession.ServiceProposal.ProviderContacts[0].Definition)
	if err != nil {
		return err
	}

	vpnConfig, err := openvpn.NewClientConfigFromString(
		vpnSession.ServiceProposal.ConnectionConfig,
		options.DirectoryRuntime+"/client.ovpn",
	)
	if err != nil {
		return err
	}

	vpnMiddlewares := append(
		cmd.vpnMiddlewares,
		bytescount_client.NewMiddleware(cmd.MysteriumClient, vpnSession.Id, 1*time.Minute),
	)
	cmd.vpnClient = openvpn.NewClient(
		vpnConfig,
		options.DirectoryRuntime,
		vpnMiddlewares...,
	)
	if err := cmd.vpnClient.Start(); err != nil {
		return err
	}

	return nil
}

func (cmd *CommandRun) Wait() error {
	return cmd.vpnClient.Wait()
}

func (cmd *CommandRun) Kill() {
	cmd.communicationClient.Stop()
	cmd.vpnClient.Stop()
}
