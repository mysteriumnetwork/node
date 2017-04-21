package command_run

import (
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/openvpn"
)

const CLIENT_NODE_KEY = "12345"

type commandRun struct {}

// NewCommand return a new instance of commandRun.
func NewCommandRun() *commandRun {
	return &commandRun{}
}

func (cmd *commandRun) Run(args ...string) error {
	mysterium := server.NewClient()
	vpnSession, err := mysterium.SessionCreate(CLIENT_NODE_KEY)
	if err != nil {
		return err
	}

	vpnConfig, err := openvpn.NewClientConfigFromString(vpnSession.ConnectionConfig)
	if err != nil {
		return err
	}

	vpnClient := openvpn.NewClient(vpnConfig)
	if err := vpnClient.Start(); err != nil {
		return err
	}

	vpnClient.Wait()
	return nil
}