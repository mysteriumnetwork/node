package openvpn

import (
	log "github.com/cihub/seelog"
)

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
	params, err := server.config.Validate()
	log.Info(SERVER_LOG_PREFIX, "Validating server params: ", params)
	if err != nil {
		return err
	}

	return server.process.Start(params...)
}

func (server *Server) Stop() (err error) {
	server.process.Stop()
	server.management.Stop()

	return
}
