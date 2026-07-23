package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	runtime_capabilities "github.com/mysteriumnetwork/runtime/capabilities"
	runtime_service "github.com/mysteriumnetwork/runtime/service"
	"github.com/pkg/errors"
)

const backendMetadataFileName = "runtime-node-services.json"

type persistedServices struct {
	Services map[string]Options `json:"services"`
}

type backendAdapter struct {
	backend      runtime_service.Backend
	metadataPath string

	mu       sync.Mutex
	services map[string]Options
}

func newBackendAdapter(backend runtime_service.Backend, runtimeDir string) Backend {
	adapter := &backendAdapter{
		backend:      backend,
		metadataPath: filepath.Join(runtimeDir, backendMetadataFileName),
		services:     make(map[string]Options),
	}
	adapter.services = adapter.loadServices()
	return adapter
}

func (adapter *backendAdapter) Create(options Options) error {
	if err := adapter.backend.Create(options.runtimeOptions()); err != nil {
		return err
	}

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	adapter.services[options.Name] = mergeOptions(adapter.services[options.Name], options)
	return adapter.persistLocked()
}

func (adapter *backendAdapter) Delete(name string) error {
	if err := adapter.backend.Delete(name); err != nil {
		return err
	}

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	delete(adapter.services, name)
	return adapter.persistLocked()
}

func (adapter *backendAdapter) Start(options Options) error {
	adapter.mu.Lock()
	merged := mergeOptions(adapter.services[options.Name], options)
	adapter.mu.Unlock()

	if err := adapter.backend.Start(merged.runtimeOptions()); err != nil {
		return err
	}

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	adapter.services[merged.Name] = merged
	return adapter.persistLocked()
}

func (adapter *backendAdapter) Stop(options Options) error {
	adapter.mu.Lock()
	merged := mergeOptions(adapter.services[options.Name], options)
	adapter.mu.Unlock()

	return adapter.backend.Stop(merged.runtimeOptions())
}

func (adapter *backendAdapter) List() ([]ServiceInfo, error) {
	runtimeServices, err := adapter.backend.List()
	if err != nil {
		return nil, err
	}

	adapter.mu.Lock()
	defer adapter.mu.Unlock()

	result := make([]ServiceInfo, 0, len(runtimeServices))
	for _, info := range runtimeServices {
		options := optionsFromRuntime(info.Options)
		if stored, exists := adapter.services[info.Name]; exists {
			options = mergeOptions(options, stored)
		}

		result = append(result, ServiceInfo{
			Name:    info.Name,
			State:   ServiceState(info.State),
			Options: options,
		})
	}

	return result, nil
}

func (adapter *backendAdapter) Capabilities() (runtime_capabilities.RuntimeCapabilities, runtime_capabilities.DetailedCapabilities) {
	return adapter.backend.Capabilities()
}

func (adapter *backendAdapter) loadServices() map[string]Options {
	data, err := os.ReadFile(adapter.metadataPath)
	if err != nil {
		return make(map[string]Options)
	}

	var persisted persistedServices
	if err := json.Unmarshal(data, &persisted); err != nil || persisted.Services == nil {
		return make(map[string]Options)
	}

	return persisted.Services
}

func (adapter *backendAdapter) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(adapter.metadataPath), 0o700); err != nil {
		return errors.Wrap(err, "failed to prepare runtime node metadata directory")
	}

	data, err := json.MarshalIndent(persistedServices{Services: adapter.services}, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to encode runtime node metadata")
	}

	if err := os.WriteFile(adapter.metadataPath, data, 0o600); err != nil {
		return errors.Wrap(err, "failed to persist runtime node metadata")
	}

	return nil
}
