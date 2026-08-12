/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

// Package deb implements the unattended updater for Debian packages.
package deb

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	PackageName            = "myst"
	RepositoryOrigin       = "LP-PPA-mysteriumnetwork-node"
	defaultHealthcheckURL  = "http://127.0.0.1:4050/healthcheck"
	defaultCommandTimeout  = 20 * time.Minute
	defaultHealthTimeout   = 2 * time.Minute
	defaultHealthInterval  = 3 * time.Second
	maxHealthResponseBytes = 64 * 1024
)

// Config controls updater behavior. Repository identity and package name are
// deliberately not configurable: accepting those values from the environment
// would turn a configuration write into root code execution.
type Config struct {
	Enabled        bool
	CheckOnly      bool
	HealthcheckURL string
	CommandTimeout time.Duration
	HealthTimeout  time.Duration
	HealthInterval time.Duration
}

// ConfigFromEnvironment reads the small set of operator-tunable settings.
func ConfigFromEnvironment(checkOnly bool) (Config, error) {
	enabled, err := envBool("MYST_UPDATER_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Enabled:        enabled,
		CheckOnly:      checkOnly,
		HealthcheckURL: envString("MYST_UPDATER_HEALTHCHECK_URL", defaultHealthcheckURL),
		CommandTimeout: defaultCommandTimeout,
		HealthTimeout:  defaultHealthTimeout,
		HealthInterval: defaultHealthInterval,
	}
	if config.CommandTimeout, err = envDuration("MYST_UPDATER_COMMAND_TIMEOUT", config.CommandTimeout); err != nil {
		return Config{}, err
	}
	if config.HealthTimeout, err = envDuration("MYST_UPDATER_HEALTHCHECK_TIMEOUT", config.HealthTimeout); err != nil {
		return Config{}, err
	}

	healthURL, err := url.Parse(config.HealthcheckURL)
	if err != nil || healthURL.Scheme != "http" || healthURL.Hostname() != "127.0.0.1" {
		return Config{}, fmt.Errorf("MYST_UPDATER_HEALTHCHECK_URL must be an http URL on 127.0.0.1")
	}
	return config, nil
}

func envString(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func envBool(name string, fallback bool) (bool, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid %s: %w", name, err)
	}
	return parsed, nil
}

func envDuration(name string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid %s duration %q", name, value)
	}
	return parsed, nil
}
