package service

import (
	"net"

	runtime_capabilities "github.com/mysteriumnetwork/runtime/capabilities"
	runtime_service "github.com/mysteriumnetwork/runtime/service"
)

type backendAdapter struct {
	backend runtime_service.Backend
}

func newBackendAdapter(backend runtime_service.Backend) Backend {
	return &backendAdapter{backend: backend}
}

func (adapter *backendAdapter) Create(options ApprovedCreateOptions) error {
	return adapter.backend.Create(options.runtimeOptions())
}

func (adapter *backendAdapter) Delete(name string) error {
	return adapter.backend.Delete(name)
}

func (adapter *backendAdapter) Start(name string) error {
	return adapter.backend.Start(name)
}

func (adapter *backendAdapter) Stop(name string) error {
	return adapter.backend.Stop(name)
}

func (adapter *backendAdapter) SetDesiredState(name string, state ServiceState) error {
	return adapter.backend.SetDesiredState(name, state)
}

func (adapter *backendAdapter) Get(name string) (ServiceInfo, bool, error) {
	services, err := adapter.List()
	if err != nil {
		return ServiceInfo{}, false, err
	}
	for _, info := range services {
		if info.Name == name {
			return info, true, nil
		}
	}
	return ServiceInfo{}, false, nil
}

func (adapter *backendAdapter) DialTCP(name string, port int) (net.Conn, error) {
	return adapter.backend.DialTCP(name, port)
}

func (adapter *backendAdapter) List() ([]ServiceInfo, error) {
	runtimeServices, err := adapter.backend.List()
	if err != nil {
		return nil, err
	}

	result := make([]ServiceInfo, 0, len(runtimeServices))
	for _, info := range runtimeServices {
		result = append(result, ServiceInfo{
			Name:    info.Name,
			State:   ServiceState(info.State),
			Desired: ServiceState(info.Desired),
			Options: optionsFromRuntime(info.Options),
		})
	}
	return result, nil
}

func (adapter *backendAdapter) Status() RuntimeStatus {
	return adapter.backend.Status()
}

func (adapter *backendAdapter) Availability() Availability {
	available, reason := adapter.backend.Availability()
	return Availability{Available: available, Reason: reason}
}

func (adapter *backendAdapter) Capabilities() (runtime_capabilities.RuntimeCapabilities, runtime_capabilities.DetailedCapabilities) {
	return adapter.backend.Capabilities()
}
