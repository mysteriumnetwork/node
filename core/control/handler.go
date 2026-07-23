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
	"fmt"
	"regexp"
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
			serviceType, err := c.deleteRuntimeService(r)
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

	if strings.HasPrefix(request.Service, "runtime.") {
		runtimeOptions, exists := c.getRuntimeService(request.Service)
		if !exists {
			if len(request.Options) == 0 {
				return errors.Errorf("runtime service %q is not created", request.Service)
			}

			runtimeInput, err := c.parseRuntimeServiceOptions(request)
			if err != nil {
				return err
			}
			runtimeOptions = toRuntimeServiceOptions(request.Service, runtimeInput)
		}

		// Optional start-time override, create remains the source of truth.
		if len(request.Options) > 0 {
			runtimeInput, err := c.parseRuntimeServiceOptions(request)
			if err != nil {
				return err
			}
			overrides := toRuntimeServiceOptions(request.Service, runtimeInput)
			if overrides.Exec != "" {
				runtimeOptions.Exec = overrides.Exec
			}
			if overrides.ServicePort > 0 {
				runtimeOptions.ServicePort = overrides.ServicePort
			}
			if len(overrides.Env) > 0 {
				runtimeOptions.Env = overrides.Env
			}
			if overrides.ResourceLimits.CPU != "" {
				runtimeOptions.ResourceLimits.CPU = overrides.ResourceLimits.CPU
			}
			if overrides.ResourceLimits.Memory != "" {
				runtimeOptions.ResourceLimits.Memory = overrides.ResourceLimits.Memory
			}
			if overrides.ResourceLimits.Disk != "" {
				runtimeOptions.ResourceLimits.Disk = overrides.ResourceLimits.Disk
			}
		}

		startRequest.Options = runtimeOptions
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
	if c.runtimeBackend == nil {
		return errors.New("runtime backend is not configured")
	}

	runtimeInput, err := c.parseRuntimeServiceOptions(request)
	if err != nil {
		return err
	}

	serviceType, err := resolveRuntimeServiceType(request.Service, runtimeInput.Name)
	if err != nil {
		return err
	}

	return c.runtimeBackend.Create(toRuntimeServiceOptions(serviceType, runtimeInput))
}

func (c *ControlPlane) deleteRuntimeService(request controlMessageItem) (string, error) {
	if c.runtimeBackend == nil {
		return "", errors.New("runtime backend is not configured")
	}

	runtimeInput, err := c.parseRuntimeServiceOptions(request)
	if err != nil {
		return "", err
	}

	serviceType, err := resolveRuntimeServiceType(request.Service, runtimeInput.Name)
	if err != nil {
		return "", err
	}

	return serviceType, c.runtimeBackend.Delete(serviceType)
}

func (c *ControlPlane) parseRuntimeServiceOptions(request controlMessageItem) (RuntimeServiceOptions, error) {
	if len(request.Options) == 0 {
		return RuntimeServiceOptions{}, nil
	}

	var options RuntimeServiceOptions
	if err := json.Unmarshal(request.Options, &options); err != nil {
		return RuntimeServiceOptions{}, errors.Wrap(err, "failed to parse runtime service options")
	}
	if options.ServicePort < 0 || options.ServicePort > 65535 {
		return RuntimeServiceOptions{}, errors.New("service_port must be between 0 and 65535")
	}

	return options, nil
}

func toRuntimeServiceOptions(serviceType string, options RuntimeServiceOptions) runtime_service_options.Options {
	return runtime_service_options.Options{
		Name:        serviceType,
		OCIArtifact: options.OCIArtifact,
		Exec:        options.Exec,
		ServicePort: options.ServicePort,
		ResourceLimits: runtime_service_options.ResourceLimits{
			CPU:    options.ResourceLimits.CPU,
			Memory: options.ResourceLimits.Memory,
			Disk:   options.ResourceLimits.Disk,
		},
	}
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

var runtimeNameSanitizer = regexp.MustCompile(`[^a-z0-9_-]+`)

func resolveRuntimeServiceType(serviceType, name string) (string, error) {
	if strings.HasPrefix(serviceType, "runtime.") {
		return serviceType, nil
	}

	if serviceType != "runtime" {
		return "", errors.Errorf("runtime command requires service runtime or runtime.<name>, got %q", serviceType)
	}

	normalized := normalizeRuntimeServiceName(name)
	if normalized == "" {
		return "", errors.New("runtime service name is required")
	}

	return fmt.Sprintf("runtime.%s", normalized), nil
}

func normalizeRuntimeServiceName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = runtimeNameSanitizer.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	return normalized
}

func validateRuntimeCommand(currentServices []contract.ServiceInfoDTO, request controlMessageItem) error {
	if !isRuntimeScopedCommand(request) {
		return nil
	}

	if hasActiveService(currentServices, runtime_service.ServiceType) {
		return nil
	}

	return errors.New("runtime service must be active to manage runtime.* services")
}

func isRuntimeScopedCommand(request controlMessageItem) bool {
	switch request.Command {
	case "create", "delete", "start", "stop":
		if strings.HasPrefix(request.Service, runtime_service.ServiceType+".") {
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
