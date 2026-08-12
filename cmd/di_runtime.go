/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

package cmd

import (
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/services"
	"github.com/mysteriumnetwork/node/services/runtime/registry"
	runtime_service_impl "github.com/mysteriumnetwork/node/services/runtime/service"
)

// runtimeServiceRegistry returns the client for the corporate registry that
// lists the workloads this node may install, or nil when none is configured.
// Nil fails every create rather than allowing one: a node without an approved
// workload list has no basis on which to admit a workload.
func (di *Dependencies) runtimeServiceRegistry() runtime_service_impl.Registry {
	address := config.GetString(config.FlagRuntimeRegistryAddress)
	if address == "" {
		log.Info().Msgf(
			"No runtime service registry configured; runtime services cannot be created (set --%s)",
			config.FlagRuntimeRegistryAddress.Name,
		)
		return nil
	}

	client, err := registry.NewClient(address)
	if err != nil {
		log.Error().Err(err).Msg("Invalid runtime service registry address; runtime services cannot be created")
		return nil
	}

	log.Info().Str("address", address).Msg("Runtime services will be installed from the service registry")
	return client
}

// reconcileRuntimeServices restores runtime-* services to the state the runtime
// backend recorded before the last shutdown. Workload state does not survive a
// node restart, so without this a service left active simply stays down.
//
// A service recorded active is started through the service manager, which is
// the same entry point a control-plane "start runtime-foo" ends up in. That is
// what registers the node's half of the service: the instance, its proposal and
// discovery, the p2p listener, and the network service that fronts the
// workload. Starting the workload alone would leave a container running that
// nothing can reach and that no operator can see in the service list.
//
// It is attempted once, when the parent runtime service comes up, and never
// fails the caller: a service that cannot be reconciled is logged and skipped
// so one broken definition cannot keep the rest down.
func (di *Dependencies) reconcileRuntimeServices(providerID identity.Identity) {
	if di.RuntimeServiceBackend == nil || di.ServicesManager == nil {
		return
	}

	runtimeServices, err := di.RuntimeServiceBackend.List()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to list runtime services; workload states were not reconciled")
		return
	}

	for _, info := range runtimeServices {
		if info.Desired != runtime_service_impl.ServiceStateActive {
			// Nothing is running node-side yet, but the workload itself can
			// have outlived an unclean shutdown of the previous process.
			if err := di.RuntimeServiceBackend.Stop(info.Name); err != nil {
				log.Warn().Err(err).Str("service", info.Name).Msg("Failed to stop passive runtime workload")
			}
			continue
		}
		di.startReconciledRuntimeService(providerID, info.Name)
	}
}

// startReconciledRuntimeService restores one service. The same service can also
// be in the configured service list and be starting concurrently, so it does
// not check first and start after: the service manager rejects the second of
// two starts atomically, and that rejection is the expected outcome here.
func (di *Dependencies) startReconciledRuntimeService(providerID identity.Identity, serviceType string) {
	serviceOpts, err := services.GetStartOptions(serviceType)
	if err != nil {
		log.Warn().Err(err).Str("service", serviceType).Msg("Failed to resolve runtime service start options")
		return
	}

	_, err = di.ServicesManager.Start(
		providerID,
		serviceType,
		serviceOpts.AccessPolicyList,
		// A start selects an installed definition and must never mutate it, so
		// the options carry nothing, exactly as a control-plane start does.
		runtime_service_impl.StartOptions{},
	)
	switch {
	case errors.Is(err, service.ErrorAlreadyRunning):
		log.Debug().Str("service", serviceType).Msg("Runtime service is already running; nothing to restore")
	case err != nil:
		log.Warn().Err(err).Str("service", serviceType).Msg("Failed to restore runtime service")
	default:
		log.Info().Str("service", serviceType).Msg("Restored runtime service that was active before restart")
	}
}

// runtimeReconciler returns the reconciler the parent runtime service runs once
// it is up, or nil when this node cannot run workloads at all.
func (di *Dependencies) runtimeReconciler() runtime_service_impl.Reconciler {
	return func(providerID identity.Identity) {
		if _, unavailable := runtime_service_impl.UnavailableReason(di.RuntimeServiceBackend); unavailable {
			return
		}
		di.reconcileRuntimeServices(providerID)
	}
}
