package openvpn

import "sync"

func NewClient(config *ClientConfig, directoryRuntime string) *Client {
	// Add the management interface socketAddress to the config
	socketAddress := tempFilename(directoryRuntime, "openvpn-management-", ".sock")
	config.SetManagementSocket(socketAddress)

	return &Client{
		config:     config,
		management: NewManagement(socketAddress, "[client-managemnet] "),
		process:    NewProcess("[client-openvpn] "),
	}
}

type Client struct {
	config     *ClientConfig
	management *Management
	process    *Process
}

func (client *Client) Start() error {
	// Start the management interface (if it isnt already started)
	err := client.management.Start()
	if err != nil {
		return err
	}

	// Fetch the current arguments
	arguments, err := ConfigToArguments(*client.config.Config)
	if err != nil {
		return err
	}

	return client.process.Start(arguments)
}

func (client *Client) Wait() error {
	return client.process.Wait()
}

func (client *Client) Stop() error {
	waiter := sync.WaitGroup{}
	go func() {
		waiter.Add(1)
		defer waiter.Done()

		client.process.Stop()
	}()
	go func() {
		waiter.Add(1)
		defer waiter.Done()

		client.management.Stop()
	}()

	waiter.Wait()
	return nil
}
