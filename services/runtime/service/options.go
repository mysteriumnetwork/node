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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mysteriumnetwork/node/core/service"
	runtime_service "github.com/mysteriumnetwork/runtime/service"
)

// CreateOptions is the complete caller-controlled input for installing a
// runtime service: which approved service to install, and optionally a
// stricter isolation floor to install it under. It deliberately cannot name an
// artifact - that comes from the service registry - and workload execution,
// networking, and resource policy are read from the immutable manifest in the
// OCI artifact.
type CreateOptions struct {
	Name                string       `json:"name,omitempty"`
	MinimumRuntimeLevel RuntimeLevel `json:"minimum_runtime_level,omitempty"`
}

// ApprovedCreateOptions is a create request that has been resolved against the
// service registry. Its artifact field is unexported so that the only way to
// obtain one is through an Installer, which makes it impossible for a caller
// outside this package to hand the backend a workload of its own choosing.
type ApprovedCreateOptions struct {
	CreateOptions
	ociArtifact string
}

// OCIArtifact is the digest-pinned artifact the registry approved.
func (options ApprovedCreateOptions) OCIArtifact() string {
	return options.ociArtifact
}

// StartOptions intentionally has no fields. A start request selects an
// installed definition; it must never mutate that definition.
type StartOptions struct{}

// Options is the normalized, immutable service definition returned by the
// runtime backend after the artifact manifest has been validated.
type Options struct {
	Name                string            `json:"name"`
	OCIArtifact         string            `json:"oci_artifact"`
	ServicePort         int               `json:"service_port"`
	Process             ProcessDefinition `json:"process"`
	ResourceLimits      ResourceLimits    `json:"resource_limits"`
	Isolation           IsolationProfile  `json:"isolation"`
	MinimumRuntimeLevel RuntimeLevel      `json:"minimum_runtime_level"`
}

// ResourceLimits describe mandatory runtime resource constraints.
type ResourceLimits = runtime_service.ResourceLimits
type ProcessDefinition = runtime_service.ProcessDefinition
type RuntimeLevel = runtime_service.RuntimeLevel
type IsolationFeatures = runtime_service.IsolationFeatures
type IsolationProfile = runtime_service.IsolationProfile

const (
	RuntimeLevelUnavailable = runtime_service.RuntimeLevelUnavailable
	RuntimeLevelUnisolated  = runtime_service.RuntimeLevelUnisolated
	RuntimeLevelLimited     = runtime_service.RuntimeLevelLimited
	RuntimeLevelFull        = runtime_service.RuntimeLevelFull
)

// GetOptions returns optionless start configuration.
func GetOptions() StartOptions {
	return StartOptions{}
}

// ParseJSONOptions parses start options and rejects every mutable field.
func ParseJSONOptions(request *json.RawMessage) (service.Options, error) {
	return ParseJSONStartOptions(request)
}

// ParseJSONStartOptions accepts only an absent, null, or empty JSON object.
func ParseJSONStartOptions(request *json.RawMessage) (service.Options, error) {
	if request == nil || len(bytes.TrimSpace(*request)) == 0 || bytes.Equal(bytes.TrimSpace(*request), []byte("null")) {
		return StartOptions{}, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(*request, &fields); err != nil {
		return StartOptions{}, err
	}
	if len(fields) != 0 {
		return StartOptions{}, fmt.Errorf("runtime start options are not supported; configure the workload in its OCI manifest")
	}
	return StartOptions{}, nil
}

// ParseJSONCreateOptions accepts only the service name and minimum runtime
// policy. Unknown fields are rejected so unsafe older contracts cannot
// silently regain effect; in particular an oci_artifact sent by a control
// message is refused rather than ignored, because a caller that believes it
// chose the workload must be told that it did not.
func ParseJSONCreateOptions(request json.RawMessage) (CreateOptions, error) {
	var options CreateOptions
	if len(bytes.TrimSpace(request)) == 0 {
		return options, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(request))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&options); err != nil {
		if strings.Contains(err.Error(), `"oci_artifact"`) {
			return options, fmt.Errorf(
				"oci_artifact cannot be set on a runtime create request; " +
					"workloads are installed by name from the service registry",
			)
		}
		return options, err
	}
	return options, nil
}

func (options ApprovedCreateOptions) runtimeOptions() runtime_service.CreateOptions {
	return runtime_service.CreateOptions{
		Name:                options.Name,
		OCIArtifact:         options.ociArtifact,
		MinimumRuntimeLevel: options.MinimumRuntimeLevel,
	}
}

func optionsFromRuntime(options runtime_service.Options) Options {
	return Options{
		Name:                options.Name,
		OCIArtifact:         options.OCIArtifact,
		ServicePort:         options.ServicePort,
		Process:             options.Process,
		ResourceLimits:      options.ResourceLimits,
		Isolation:           options.Isolation,
		MinimumRuntimeLevel: options.MinimumRuntimeLevel,
	}
}
