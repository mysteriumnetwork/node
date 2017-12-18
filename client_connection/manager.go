package client_connection

import "github.com/mysterium/node/service_discovery/dto"

type vpnManager struct {
}

func NewVpnManager() *vpnManager {

	return &vpnManager{}
}

func (vpn *vpnManager) Connect(identity dto.Identity, NodeKey string) error {
	return nil
}

func (vpn *vpnManager) Status() ConnectionStatus {
	return ConnectionStatus{}
}

func (vpn *vpnManager) Disconnect() error {
	return nil
}

func (vpn *vpnManager) Wait() error {
	return nil
}
