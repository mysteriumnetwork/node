package openvpn

import (
	log "github.com/cihub/seelog"
)

const CLIENT_LOG_PREFIX = "[OpenVPN.process] "

func NewClient(config *ClientConfig) *Client {
	return &Client{
		config: config,
		management: NewManagement(),
		process: NewProcess(CLIENT_LOG_PREFIX),
	}
}

type Client struct {
	config *ClientConfig
	management *Management
	process *Process
}

func (client *Client) Start() (err error) {
	// Start the management interface (if it isnt already started)
	path, err := client.management.Start()
	if err != nil {
		return err
	}

	// Add the management interface path to the config
	client.config.SetManagementPath(path)

	// Fetch the current params
	params, err := client.config.Validate()
	log.Info(CLIENT_LOG_PREFIX, "Validating process params: ", params)
	if err != nil {
		return err
	}

	return client.process.Start(params...)
}

func (client *Client) Wait() {
	client.process.Wait()
}

func (client *Client) Stop() (err error) {
	client.process.Stop()
	client.management.Stop()

	return
}
