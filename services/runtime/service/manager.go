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
	"github.com/mysteriumnetwork/node/identity"
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

// ServiceInfo describes a runtime service. State is what the workload is doing
// now; Desired is the intent recorded by the runtime backend, which survives a
// node restart and drives what gets started again on boot.
type ServiceInfo struct {
	Name    string       `json:"name"`
	State   ServiceState `json:"state"`
	Desired ServiceState `json:"desired_state"`
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
	// Create installs a definition that has been approved against the service
	// registry. Its argument can only be produced by an Installer, so no
	// caller can install a workload the registry does not list.
	Create(options ApprovedCreateOptions) error
	Delete(name string) error
	Start(name string) error
	Stop(name string) error
	// SetDesiredState records whether a service should be running. Stop only
	// stops the workload; it does not mean the service should stay down.
	SetDesiredState(name string, state ServiceState) error
	Get(name string) (ServiceInfo, bool, error)
	DialTCP(name string, port int) (net.Conn, error)
	List() ([]ServiceInfo, error)
	Status() RuntimeStatus
	Availability() Availability
	Capabilities() (runtime_capabilities.RuntimeCapabilities, runtime_capabilities.DetailedCapabilities)
}

// Reconciler restores runtime workloads to their persisted desired state. It
// runs once, when the parent runtime service is up, and is given that service's
// provider identity so it can start runtime-* services the same way a
// control-plane start does.
//
// The stopped channel is closed when the parent runtime service stops. Work the
// reconciler keeps doing afterwards - the periodic sync against the service
// registry - is bounded by it, so nothing keeps managing workloads once the
// service that fronts them is gone.
type Reconciler func(providerID identity.Identity, stopped <-chan struct{})

type Manager struct {
	backend        Backend
	name           string
	networkService service.Service
	reconcile      Reconciler
	// shuttingDown reports whether the node process itself is going down. A
	// service stopped because the node is exiting must come back on the next
	// start; one an operator stopped must not.
	shuttingDown  func() bool
	isBaseRuntime bool

	stateMu       sync.Mutex
	stopRequested bool
	started       bool

	stopOnce sync.Once
	done     chan struct{}
}

func NewManager(
	backend Backend,
	name string,
	networkService service.Service,
	reconcile Reconciler,
	shuttingDown func() bool,
) *Manager {
	return &Manager{
		backend:        backend,
		name:           name,
		networkService: networkService,
		reconcile:      reconcile,
		shuttingDown:   shuttingDown,
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
			// Serve runs after the parent runtime service is registered, so
			// this is the first point where runtime-* services can be started
			// through the node the way the control plane starts them.
			if manager.reconcile != nil {
				manager.reconcile(instance.ProviderID, manager.done)
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

	if err := manager.backend.Stop(name); err != nil {
		return err
	}
	// Both a node shutdown and an operator stop reach this same call, so the
	// backend cannot tell them apart. Only the latter means the service should
	// stay down across a restart.
	if manager.shuttingDown != nil && manager.shuttingDown() {
		return nil
	}
	return manager.backend.SetDesiredState(name, ServiceStatePassive)
}
