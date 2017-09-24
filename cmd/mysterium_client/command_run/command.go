package command_run

import (
	"github.com/mysterium/node/bytescount_client"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/service_discovery"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"io"
	"time"
)

type commandRun struct {
	output      io.Writer
	outputError io.Writer

	mysteriumClient     server.Client
	communicationClient communication.Client

	vpnMiddlewares []openvpn.ManagementMiddleware
	vpnClient      *openvpn.Client
}

func (cmd *commandRun) Run(options CommandOptions) (err error) {
	providerId := dto_discovery.Identity(options.NodeKey)
	serviceProposal := service_discovery.NewServiceProposal(providerId, nats.NewContact(providerId))

	_, _, err = cmd.communicationClient.CreateDialog(serviceProposal.ProviderContacts[0])
	if err != nil {
		return err
	}

	vpnSession, err := cmd.mysteriumClient.SessionCreate(options.NodeKey)
	if err != nil {
		return err
	}

	vpnConfig, err := openvpn.NewClientConfigFromString(
		vpnSession.ConnectionConfig,
		options.DirectoryRuntime+"/client.ovpn",
	)
	if err != nil {
		return err
	}

	vpnMiddlewares := append(
		cmd.vpnMiddlewares,
		bytescount_client.NewMiddleware(cmd.mysteriumClient, vpnSession.Id, 1*time.Minute),
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

func (cmd *commandRun) Wait() error {
	return cmd.vpnClient.Wait()
}

func (cmd *commandRun) Kill() {
	cmd.communicationClient.Stop()
	cmd.vpnClient.Stop()
}
