/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package control

import (
	"encoding/json"
	"strings"

	"github.com/mysteriumnetwork/node/services"
	runtime_service "github.com/mysteriumnetwork/node/services/runtime"
	runtime_service_options "github.com/mysteriumnetwork/node/services/runtime/service"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// handler is a function that handles control messages
func (c *ControlPlane) handler(request controlMessage) error {
	currentServices, err := c.api.Services()
	if err != nil {
		return err
	}

	var firstErr error

	for _, r := range request {
		log.Info().Str("command", r.Command).Str("service", r.Service).Msg("executing control request")

		if requiresRuntimeAvailability(r) {
			if err := c.ensureRuntimeAvailable(); err != nil {
				log.Warn().AnErr("err", err).Msg("runtime control request rejected")
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
		}
		if isRuntimeCommand(r) && (r.Command == "start" || r.Command == "stop" || r.Command == "restart") {
			if err := rejectRuntimeCommandOptions(r); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
		}
		if err := validateRuntimeCommand(currentServices, r); err != nil {
			log.Warn().AnErr("err", err).Msg("runtime control request rejected")
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		switch r.Command {
		case "create":
			if err := c.createRuntimeService(r); err != nil {
				log.Warn().AnErr("err", err).Msg("failed to create runtime service")
				if firstErr == nil {
					firstErr = err
				}
			}
		case "delete":
			serviceType, err := c.resolveRuntimeServiceForDelete(r)
			if err != nil {
				log.Warn().AnErr("err", err).Msg("failed to delete runtime service")
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			for _, service := range currentServices {
				if service.Type != serviceType {
					continue
				}
				if err := c.stopService(service.ID); err != nil {
					log.Warn().AnErr("err", err).Msg("failed to stop runtime service on delete")
					if firstErr == nil {
						firstErr = err
					}
				}
			}
			if err := c.runtimeBackend.Delete(serviceType); err != nil {
				log.Warn().AnErr("err", err).Msg("failed to delete runtime service definition")
				if firstErr == nil {
					firstErr = err
				}
			}
		case "start":
			if err := c.startService(r); err != nil {
				log.Warn().AnErr("err", err).Msg("failed to start service")
				if firstErr == nil {
					firstErr = err
				}
			}
		case "stop":
			for _, service := range currentServices {
				if service.Type != r.Service {
					continue
				}
				if err := c.stopService(service.ID); err != nil {
					log.Warn().AnErr("err", err).Msg("failed to stop service")
					if firstErr == nil {
						firstErr = err
					}
				}
			}
		case "restart":
			for _, service := range currentServices {
				if service.Type != r.Service {
					continue
				}
				if err := c.stopService(service.ID); err != nil {
					log.Warn().AnErr("err", err).Msg("failed to stop service")
					if firstErr == nil {
						firstErr = err
					}
				}
				if err := c.startService(r); err != nil {
					log.Warn().AnErr("err", err).Msg("failed to start service")
					if firstErr == nil {
						firstErr = err
					}
				}
			}
		default:
			log.Warn().Str("command", r.Command).Msg("unknown control command")
			if firstErr == nil {
				firstErr = errors.Errorf("unknown control command %q", r.Command)
			}
		}
	}

	return firstErr
}

func (c *ControlPlane) startService(request controlMessageItem) error {
	if request.Service == "" {
		return errors.New("service is required")
	}

	serviceOpts, err := services.GetStartOptions(request.Service)
	if err != nil {
		return err
	}

	providerID := request.ProviderID
	if providerID == "" {
		providerID = c.identity
	}

	startRequest := contract.ServiceStartRequest{
		ProviderID: providerID,
		Type:       request.Service,
		Options:    serviceOpts,
	}

	if len(serviceOpts.AccessPolicyList) > 0 {
		startRequest.AccessPolicies = &contract.ServiceAccessPolicies{IDs: serviceOpts.AccessPolicyList}
	}
	if len(request.AccessPolicies) > 0 {
		startRequest.AccessPolicies = &contract.ServiceAccessPolicies{IDs: request.AccessPolicies}
	}

	if strings.HasPrefix(request.Service, runtime_service.ServiceTypePrefix) {
		_, exists := c.getRuntimeService(request.Service)
		if !exists {
			return errors.Errorf("runtime service %q is not created", request.Service)
		}
		startRequest.Options = runtime_service_options.StartOptions{}
	} else if len(request.Options) > 0 {
		startRequest.Options = json.RawMessage(request.Options)
	} else {
		startRequest.Options = serviceOpts
	}

	_, err = c.api.ServiceStart(startRequest)
	return err
}

func (c *ControlPlane) stopService(id string) error {
	return c.api.ServiceStop(id)
}

func (c *ControlPlane) createRuntimeService(request controlMessageItem) error {
	if c.runtimeInstaller == nil {
		return errors.New("runtime service registry is not configured; runtime services cannot be created")
	}

	runtimeInput, err := c.parseRuntimeCreateOptions(request)
	if err != nil {
		return err
	}

	serviceType, err := resolveRuntimeServiceType(request.Service, runtimeInput.Name)
	if err != nil {
		return err
	}

	// The installer resolves this name against the service registry; a name it
	// does not publish never reaches the runtime backend.
	return c.runtimeInstaller.Install(toRuntimeServiceOptions(serviceType, runtimeInput))
}

func (c *ControlPlane) resolveRuntimeServiceForDelete(request controlMessageItem) (string, error) {
	if c.runtimeBackend == nil {
		return "", errors.New("runtime backend is not configured")
	}

	name, err := parseRuntimeDeleteName(request)
	if err != nil {
		return "", err
	}
	serviceType, err := resolveRuntimeServiceType(request.Service, name)
	if err != nil {
		return "", err
	}

	return serviceType, nil
}

// parseRuntimeCreateOptions reads the create contract. Options may be absent
// entirely when the request names the service as runtime-<name>: with the
// artifact coming from the registry, the name is all a create needs.
func (c *ControlPlane) parseRuntimeCreateOptions(request controlMessageItem) (RuntimeServiceOptions, error) {
	options, err := runtime_service_options.ParseJSONCreateOptions(request.Options)
	if err != nil {
		return RuntimeServiceOptions{}, errors.Wrap(err, "failed to parse runtime service options")
	}
	return options, nil
}

func toRuntimeServiceOptions(serviceType string, options RuntimeServiceOptions) runtime_service_options.CreateOptions {
	options.Name = serviceType
	return options
}

func (c *ControlPlane) getRuntimeService(serviceType string) (runtime_service_options.Options, bool) {
	if c.runtimeBackend == nil {
		return runtime_service_options.Options{}, false
	}

	services, err := c.runtimeBackend.List()
	if err != nil {
		log.Warn().Err(err).Str("service", serviceType).Msg("failed to list runtime services")
		return runtime_service_options.Options{}, false
	}

	for _, serviceInfo := range services {
		if serviceInfo.Name == serviceType {
			return serviceInfo.Options, true
		}
	}

	return runtime_service_options.Options{}, false
}

func resolveRuntimeServiceType(serviceType, name string) (string, error) {
	if strings.HasPrefix(serviceType, runtime_service.ServiceTypePrefix) {
		return serviceType, nil
	}

	if serviceType != runtime_service.ServiceType {
		return "", errors.Errorf("runtime command requires service runtime or runtime-<name>, got %q", serviceType)
	}

	// Normalized the same way the service registry indexes its entries, so a
	// service is reachable by the name an operator reads there.
	normalized := runtime_service.NormalizeServiceType(name)
	if normalized == "" {
		return "", errors.New("runtime service name is required")
	}

	return normalized, nil
}

func validateRuntimeCommand(currentServices []contract.ServiceInfoDTO, request controlMessageItem) error {
	if !isRuntimeScopedCommand(request) {
		return nil
	}

	if hasActiveService(currentServices, runtime_service.ServiceType) {
		return nil
	}

	return errors.New("runtime service must be active to manage runtime-* services")
}

func (c *ControlPlane) ensureRuntimeAvailable() error {
	reason, unavailable := runtime_service_options.UnavailableReason(c.runtimeBackend)
	if !unavailable {
		return nil
	}
	return errors.Errorf("runtime service is unavailable: %s", reason)
}

func isRuntimeCommand(request controlMessageItem) bool {
	return request.Service == runtime_service.ServiceType ||
		strings.HasPrefix(request.Service, runtime_service.ServiceTypePrefix)
}

func requiresRuntimeAvailability(request controlMessageItem) bool {
	if !isRuntimeCommand(request) {
		return false
	}
	switch request.Command {
	case "create", "start", "restart":
		return true
	default:
		return false
	}
}

func rejectRuntimeCommandOptions(request controlMessageItem) error {
	if len(request.Options) == 0 {
		return nil
	}
	_, err := runtime_service_options.ParseJSONStartOptions(&request.Options)
	return err
}

func parseRuntimeDeleteName(request controlMessageItem) (string, error) {
	if strings.HasPrefix(request.Service, runtime_service.ServiceTypePrefix) {
		if err := rejectRuntimeCommandOptions(request); err != nil {
			return "", err
		}
		return "", nil
	}
	if len(request.Options) == 0 {
		return "", errors.New("runtime service name is required")
	}
	var selector struct {
		Name string `json:"name"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(request.Options)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&selector); err != nil {
		return "", errors.Wrap(err, "failed to parse runtime delete selector")
	}
	return selector.Name, nil
}

func isRuntimeScopedCommand(request controlMessageItem) bool {
	switch request.Command {
	case "create", "delete", "start", "stop", "restart":
		if strings.HasPrefix(request.Service, runtime_service.ServiceTypePrefix) {
			return true
		}

		return (request.Command == "create" || request.Command == "delete") && request.Service == runtime_service.ServiceType
	default:
		return false
	}
}

func hasActiveService(services []contract.ServiceInfoDTO, serviceType string) bool {
	for _, service := range services {
		if service.Type == serviceType {
			return true
		}
	}

	return false
}
