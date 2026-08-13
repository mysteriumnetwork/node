/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package service

import (
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/service"
	runtime_service "github.com/mysteriumnetwork/node/services/runtime"
	"github.com/mysteriumnetwork/node/services/runtime/registry"
)

const (
	// DefaultReconcileInterval is how often the installed workloads are
	// compared against the registry.
	DefaultReconcileInterval = time.Hour
	// DefaultReconcileMissThreshold is how many consecutive syncs must agree
	// that a service is gone before it is removed.
	DefaultReconcileMissThreshold = 3
)

// InstanceLifecycle is the node-side half of a runtime service: the instance,
// its proposal and discovery. It is satisfied by the service manager.
type InstanceLifecycle interface {
	List(includeAll bool) []*service.Instance
	Stop(id service.ID) error
}

// RegistryReconciler removes workloads the registry no longer approves. The
// registry is the only authority on what a node may run, so a service dropped
// from it has to stop running here too - but an absence has to be believable
// first, which is what makes this a periodic count rather than a single check.
//
// Counts are kept in memory only. A node that restarts starts counting again,
// which delays a removal by at most one interval per restart and never causes
// one that the registry did not ask for.
type RegistryReconciler struct {
	backend   Backend
	registry  Registry
	instances InstanceLifecycle
	interval  time.Duration
	threshold int

	mu sync.Mutex
	// misses counts, per service type, how many consecutive successful syncs
	// have not approved its installed artifact. Only syncs that reached a usable
	// registry definition are counted.
	misses map[string]int
}

// NewRegistryReconciler creates a reconciler that removes a service after
// DefaultReconcileMissThreshold consecutive syncs without approval for its
// installed artifact.
func NewRegistryReconciler(backend Backend, serviceRegistry Registry, instances InstanceLifecycle) *RegistryReconciler {
	return &RegistryReconciler{
		backend:   backend,
		registry:  serviceRegistry,
		instances: instances,
		interval:  DefaultReconcileInterval,
		threshold: DefaultReconcileMissThreshold,
		misses:    make(map[string]int),
	}
}

// Run reconciles once per interval until stopped is closed. It does not sync on
// entry: the first sync of a node's life carries no more information than any
// other, and waiting one interval keeps a restart loop from counting quickly.
func (reconciler *RegistryReconciler) Run(stopped <-chan struct{}) {
	ticker := time.NewTicker(reconciler.interval)
	defer ticker.Stop()

	log.Info().
		Dur("interval", reconciler.interval).
		Int("threshold", reconciler.threshold).
		Msg("Reconciling installed runtime services against the service registry")

	for {
		select {
		case <-stopped:
			return
		case <-ticker.C:
			reconciler.Reconcile()
		}
	}
}

// Reconcile runs one sync. It never reports an error, because there is no
// caller that could act on one: everything it can fail at is retried by the
// next sync, and until then the node keeps running what it is running.
func (reconciler *RegistryReconciler) Reconcile() {
	listed, err := reconciler.registry.ListServices()
	if err != nil {
		// An unreachable registry, or one that does not answer 200 OK, says
		// nothing about which workloads are approved. The sync is skipped
		// whole: nothing is removed, and the absence is not counted against any
		// service, so an outage cannot age a working service out.
		log.Warn().Err(err).Msg("Runtime service registry is unavailable; installed services were left untouched")
		return
	}

	installed, err := reconciler.backend.List()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to list installed runtime services; nothing was reconciled")
		return
	}

	reconciler.mu.Lock()
	defer reconciler.mu.Unlock()

	// Counts are rebuilt from what is installed now, so a service that is
	// listed again, or that goes away by another route, drops its history
	// instead of carrying a stale count into a later absence.
	misses := make(map[string]int, len(reconciler.misses))
	for _, info := range installed {
		// The parent runtime service is node configuration rather than a
		// workload the registry publishes, so it is never a removal candidate.
		if info.Name == runtime_service.ServiceType {
			continue
		}
		entry, selectionErr := registry.SelectService(listed, info.Name)
		if selectionErr != nil && !errors.Is(selectionErr, registry.ErrNotListed) {
			// An invalid or ambiguous definition does not prove that the service
			// was withdrawn. Leave it untouched until the registry is repaired.
			log.Warn().Err(selectionErr).Str("service", info.Name).
				Msg("Runtime service registry definition is unusable; installed service was left untouched")
			continue
		}

		reason := "service is not listed"
		if selectionErr == nil {
			artifact, artifactErr := entry.ArtifactFor(runtime.GOOS, runtime.GOARCH)
			if artifactErr != nil {
				log.Warn().Err(artifactErr).Str("service", info.Name).
					Msg("Runtime service registry definition is unusable; installed service was left untouched")
				continue
			}
			if info.Options.OCIArtifact == artifact {
				continue
			}
			reason = "approved artifact changed"
		}

		missed := reconciler.misses[info.Name] + 1
		if missed < reconciler.threshold {
			log.Info().
				Str("service", info.Name).
				Str("reason", reason).
				Int("missed_syncs", missed).
				Int("threshold", reconciler.threshold).
				Msg("Installed runtime service does not match the service registry")
			misses[info.Name] = missed
			continue
		}

		if err := reconciler.remove(info.Name); err != nil {
			// The count is kept at the threshold so the next sync retries the
			// removal instead of granting the service another full round.
			misses[info.Name] = missed
			log.Warn().Err(err).Str("service", info.Name).
				Msg("Failed to remove runtime service that the service registry no longer approves")
			continue
		}
		log.Info().Str("service", info.Name).Int("missed_syncs", missed).
			Msg("Removed runtime service that the service registry no longer approves")
	}
	reconciler.misses = misses
}

// remove takes the node-side service down before deleting the definition: a
// definition removed underneath a running instance would leave a published
// proposal for a workload that no longer exists.
func (reconciler *RegistryReconciler) remove(serviceType string) error {
	if reconciler.instances != nil {
		for _, instance := range reconciler.instances.List(false) {
			if instance == nil || instance.Type != serviceType {
				continue
			}
			if err := reconciler.instances.Stop(instance.ID); err != nil {
				return errors.Wrapf(err, "failed to stop runtime service %q", serviceType)
			}
		}
	}
	return reconciler.backend.Delete(serviceType)
}
