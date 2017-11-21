package command_run

import (
	"github.com/mysterium/node/bytescount_client"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"io"
	"time"
	"encoding/json"
	vpn_session "github.com/mysterium/node/openvpn/session"
	"fmt"
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

	vpnSessionJson, err := sender.Request(communication.SESSION_CREATE, string(proposal.Id))
	if err != nil {
		return err
	}

	vpnSession := vpn_session.VpnSession{}
	err = json.Unmarshal([]byte(vpnSessionJson), &vpnSession)
	if err != nil {
		return err
	}

	session.Id = string(vpnSession.Id)

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

	return nil
}

func (cmd *CommandRun) Wait() error {
	return cmd.vpnClient.Wait()
}

func (cmd *CommandRun) Kill() {
	cmd.communicationClient.Stop()
	cmd.vpnClient.Stop()
}
