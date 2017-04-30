package openvpn

import "sync"

func NewServer(config *ServerConfig, directoryRuntime string) *Server {
	// Add the management interface socketAddress to the config
	socketAddress := tempFilename(directoryRuntime, "openvpn-management-", ".sock")
	config.SetManagementSocket(socketAddress)

	return &Server{
		config:     config,
		management: NewManagement(socketAddress, "[server-managemnet] "),
		process:    NewProcess("[server-openvpn] "),
	}
}

type Server struct {
	config     *ServerConfig
	management *Management
	process    *Process
}

func (server *Server) Start() error {
	// Start the management interface (if it isnt already started)
	if err := server.management.Start(); err != nil {
		return err
	}

	// Fetch the current params
	arguments, err := ConfigToArguments(*server.config.Config)
	if err != nil {
		return err
	}

	return server.process.Start(arguments)
}

func (client *Server) Wait() {
	client.process.Wait()
}

func (server *Server) Stop() {
	waiter := sync.WaitGroup{}
	go func() {
		waiter.Add(1)
		defer waiter.Done()

		server.process.Stop()
	}()
	go func() {
		waiter.Add(1)
		defer waiter.Done()

		server.management.Stop()
	}()

	waiter.Wait()
}
