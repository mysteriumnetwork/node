package command_run

import (
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/openvpn"
	"io"
)

const CLIENT_NODE_KEY = "12345"

type commandRun struct {
	output      io.Writer
	outputError io.Writer
}

func (cmd *commandRun) Run(args []string) error {
	options, err := cmd.parseArguments(args)
	if err != nil {
		return err
	}

	mysterium := server.NewClient()
	vpnSession, err := mysterium.SessionCreate(CLIENT_NODE_KEY)
	if err != nil {
		return err
	}

	vpnConfig, err := openvpn.NewClientConfigFromString(
		vpnSession.ConnectionConfig,
		options.DirectoryRuntime+ "/client.ovpn",
	)
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