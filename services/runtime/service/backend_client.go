package service

import (
	runtime_service "github.com/mysteriumnetwork/runtime/service"
)

// NewBackend creates the default runtime backend implementation.
func NewBackend(dataDir string) Backend {
	return newBackendAdapter(runtime_service.NewBackend(dataDir))
}
