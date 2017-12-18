package client_connection

import (
	"github.com/mysterium/node/bytescount_client"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/openvpn"
	vpn_session "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/service_discovery/dto"
	"path/filepath"
	"time"
)

type DialogEstablisherFactory func(identity dto.Identity) communication.DialogEstablisher

type vpnManager struct {
	//these are passed on creation
	mysteriumClient            server.Client
	dialogEstablisherFactory   func(identity dto.Identity) communication.DialogEstablisher
	vpnMiddlewares             []openvpn.ManagementMiddleware
	runtimeDirectory           string
	vpnClientConfigurationPath string
	//these are created by Connect at runtime
	dialog    communication.Dialog
	vpnClient *openvpn.Client
}

func NewVpnManager(mysteriumClient server.Client, dialogEstablisherFactory DialogEstablisherFactory, runtimeDirectory string) *vpnManager {
	vpnClientConfigFile := filepath.Join(runtimeDirectory, "client.ovpn")
	return &vpnManager{
		mysteriumClient,
		dialogEstablisherFactory,
		[]openvpn.ManagementMiddleware{},
		runtimeDirectory,
		vpnClientConfigFile,
		nil,
		nil,
	}
}

func (vpn *vpnManager) Connect(identity dto.Identity, NodeKey string) error {

	session, err := vpn.mysteriumClient.SessionCreate(NodeKey)
	if err != nil {
		return err
	}
	proposal := session.ServiceProposal

	dialogEstablisher := vpn.dialogEstablisherFactory(identity)
	vpn.dialog, err = dialogEstablisher.CreateDialog(proposal.ProviderContacts[0])
	if err != nil {
		return err
	}

	vpnSession, err := vpn_session.RequestSessionCreate(vpn.dialog, proposal.Id)
	if err != nil {
		return err
	}

	vpnConfig, err := openvpn.NewClientConfigFromString(
		vpnSession.Config,
		vpn.vpnClientConfigurationPath,
	)
	if err != nil {
		return err
	}

	vpnMiddlewares := append(
		vpn.vpnMiddlewares,
		bytescount_client.NewMiddleware(vpn.mysteriumClient, session.Id, 1*time.Minute),
	)
	vpn.vpnClient = openvpn.NewClient(
		vpnConfig,
		vpn.runtimeDirectory,
		vpnMiddlewares...,
	)
	if err := vpn.vpnClient.Start(); err != nil {
		return err
	}

	return nil
}

func (vpn *vpnManager) Status() ConnectionStatus {
	return ConnectionStatus{}
}

func (vpn *vpnManager) Disconnect() error {
	vpn.dialog.Close()
	vpn.vpnClient.Stop()
	return nil
}

func (vpn *vpnManager) Wait() error {
	return vpn.vpnClient.Wait()
}
