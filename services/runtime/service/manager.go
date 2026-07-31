/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

package service

import (
	"encoding/json"
	"net"
	"strings"
	"sync"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/p2p"
	runtime_service "github.com/mysteriumnetwork/node/services/runtime"
	runtime_capabilities "github.com/mysteriumnetwork/runtime/capabilities"
	runtime_lib_service "github.com/mysteriumnetwork/runtime/service"
	"github.com/pkg/errors"
)

type ServiceState = runtime_lib_service.ServiceState

const (
	ServiceStateActive  ServiceState = runtime_lib_service.ServiceStateActive
	ServiceStatePassive ServiceState = runtime_lib_service.ServiceStatePassive
)

type ServiceInfo struct {
	Name    string       `json:"name"`
	State   ServiceState `json:"state"`
	Options Options      `json:"options,omitempty"`
}

type Availability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

type RuntimeStatus = runtime_lib_service.RuntimeStatus

// UnavailableReason reports whether the backend's runtime status prevents all
// workload execution. Callers should use this before exposing a runtime-backed
// service to discovery.
func UnavailableReason(backend Backend) (string, bool) {
	if backend == nil {
		return "runtime backend is not configured", true
	}

	status := backend.Status()
	if status.Level != RuntimeLevelUnavailable {
		return "", false
	}

	reason := strings.Join(status.BlockingReasons, ", ")
	if reason == "" {
		reason = "required host isolation is unavailable"
	}
	return reason, true
}

type Backend interface {
	Create(options CreateOptions) error
	Delete(name string) error
	Start(name string) error
	Stop(name string) error
	Get(name string) (ServiceInfo, bool, error)
	DialTCP(name string, port int) (net.Conn, error)
	List() ([]ServiceInfo, error)
	Status() RuntimeStatus
	Availability() Availability
	Capabilities() (runtime_capabilities.RuntimeCapabilities, runtime_capabilities.DetailedCapabilities)
}

type Manager struct {
	backend        Backend
	name           string
	networkService service.Service
	isBaseRuntime  bool

	stateMu       sync.Mutex
	stopRequested bool
	started       bool

	stopOnce sync.Once
	done     chan struct{}
}

func NewManager(backend Backend, name string, networkService service.Service) *Manager {
	return &Manager{
		backend:        backend,
		name:           name,
		networkService: networkService,
		done:           make(chan struct{}),
	}
}

func (manager *Manager) ProvideConfig(sessionID string, sessionConfig json.RawMessage, conn p2p.ServiceConn) (*service.ConfigParams, error) {
	if manager.networkService != nil {
		return manager.networkService.ProvideConfig(sessionID, sessionConfig, conn)
	}
	return &service.ConfigParams{}, nil
}

// Serve starts exactly the immutable definition selected by the instance type.
// A concurrent Stop is remembered before Start returns and triggers immediate
// cleanup, preventing the workload from being orphaned.
func (manager *Manager) Serve(instance *service.Instance) error {
	name := manager.name
	if instance != nil {
		if instance.Type == runtime_service.ServiceType {
			manager.stateMu.Lock()
			manager.isBaseRuntime = true
			stopped := manager.stopRequested
			manager.stateMu.Unlock()
			if stopped {
				return nil
			}
			<-manager.done
			return nil
		}
		if name == "" && strings.HasPrefix(instance.Type, runtime_service.ServiceTypePrefix) {
			name = instance.Type
		}
	}
	if name == "" {
		return errors.New("runtime service name is required")
	}

	manager.stateMu.Lock()
	if manager.stopRequested {
		manager.stateMu.Unlock()
		return nil
	}
	manager.stateMu.Unlock()

	if err := manager.backend.Start(name); err != nil {
		return err
	}

	manager.stateMu.Lock()
	manager.name = name
	manager.started = true
	stopped := manager.stopRequested
	manager.stateMu.Unlock()

	defer manager.stopWorkload()
	if stopped {
		return nil
	}
	if manager.networkService != nil {
		return manager.networkService.Serve(instance)
	}
	<-manager.done
	return nil
}

func (manager *Manager) Stop() error {
	manager.stateMu.Lock()
	manager.stopRequested = true
	base := manager.isBaseRuntime
	manager.stateMu.Unlock()

	manager.stopOnce.Do(func() { close(manager.done) })

	var firstErr error
	if manager.networkService != nil && !base {
		firstErr = manager.networkService.Stop()
	}
	if err := manager.stopWorkload(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (manager *Manager) stopWorkload() error {
	manager.stateMu.Lock()
	if !manager.started {
		manager.stateMu.Unlock()
		return nil
	}
	manager.started = false
	name := manager.name
	manager.stateMu.Unlock()
	return manager.backend.Stop(name)
}

func (manager *Manager) List() ([]ServiceInfo, error) {
	if manager.backend == nil {
		return nil, nil
	}
	return manager.backend.List()
}

func (manager *Manager) Status() RuntimeStatus {
	if manager.backend == nil {
		return RuntimeStatus{
			Level:           RuntimeLevelUnavailable,
			BlockingReasons: []string{"runtime backend is not configured"},
		}
	}
	return manager.backend.Status()
}

func (manager *Manager) Capabilities() (runtime_capabilities.RuntimeCapabilities, runtime_capabilities.DetailedCapabilities) {
	if manager.backend == nil {
		return runtime_capabilities.Detect()
	}
	return manager.backend.Capabilities()
}
