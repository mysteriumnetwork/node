package openvpn

const SERVER_LOG_PREFIX = "[OpenVPN.server] "

func NewServer(config *ServerConfig) *Server {
	return &Server{
		config: config,
		management: NewManagement(),
		process: NewProcess(SERVER_LOG_PREFIX),
	}
}

type Server struct {
	config *ServerConfig
	management *Management
	process *Process
}

func (server *Server) Start() (err error) {
	// Start the management interface (if it isnt already started)
	path, err := server.management.Start()
	if err != nil {
		return err
	}

	// Add the management interface path to the config
	server.config.SetManagementPath(path)

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

func (server *Server) Stop() (err error) {
	server.process.Stop()
	server.management.Stop()

	return
}
