/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

package deb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type commandRunner interface {
	Output(ctx context.Context, name string, args ...string) (string, error)
	Run(ctx context.Context, name string, args ...string) error
}

type systemRunner struct{}

func (systemRunner) Output(ctx context.Context, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Env = secureEnvironment()
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s failed: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func (systemRunner) Run(ctx context.Context, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Env = secureEnvironment()
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}

func secureEnvironment() []string {
	return []string{
		"PATH=/usr/sbin:/usr/bin:/sbin:/bin",
		"LC_ALL=C",
		"LANG=C",
		"DEBIAN_FRONTEND=noninteractive",
	}
}

// Updater validates and installs a newer myst package through APT.
type Updater struct {
	config Config
	runner commandRunner
	client *http.Client
}

// New constructs a production updater.
func New(config Config) *Updater {
	return &Updater{
		config: config,
		runner: systemRunner{},
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

var aptSecurityOptions = []string{
	"-o", "Acquire::AllowInsecureRepositories=false",
	"-o", "Acquire::AllowDowngradeToInsecureRepositories=false",
	"-o", "Acquire::Check-Valid-Until=true",
	"-o", "Acquire::https::Verify-Peer=true",
	"-o", "Acquire::https::Verify-Host=true",
	"-o", "APT::Get::AllowUnauthenticated=false",
	"-o", "DPkg::Lock::Timeout=300",
}

// Run performs a single update check and, when eligible, installs the update.
func (updater *Updater) Run(ctx context.Context) error {
	if !updater.config.Enabled {
		log.Print("automatic updates are disabled")
		return nil
	}

	commandContext, cancel := context.WithTimeout(ctx, updater.config.CommandTimeout)
	defer cancel()

	if err := updater.aptGet(commandContext, "update"); err != nil {
		return fmt.Errorf("could not refresh authenticated APT metadata: %w", err)
	}

	repositoryPolicy, err := updater.runner.Output(commandContext, "apt-cache", "policy")
	if err != nil {
		return err
	}
	if err := validateRepositoryPolicy(repositoryPolicy); err != nil {
		return err
	}

	packagePolicy, err := updater.runner.Output(commandContext, "apt-cache", "policy", packageName)
	if err != nil {
		return err
	}
	candidate, err := parseCandidate(packagePolicy)
	if err != nil {
		return err
	}
	if err := validateCandidateSource(packagePolicy, candidate); err != nil {
		return err
	}

	installed, err := updater.installedVersion(commandContext)
	if err != nil {
		return err
	}
	if candidate == installed {
		log.Printf("%s is current at version %s", packageName, installed)
		return nil
	}

	newer, err := updater.versionIsGreater(commandContext, candidate, installed)
	if err != nil {
		return err
	}
	if !newer {
		log.Printf("refusing downgrade from %s to %s", installed, candidate)
		return nil
	}
	held, err := updater.packageIsHeld(commandContext)
	if err != nil {
		return err
	}
	if held {
		log.Printf("%s is held; leaving version %s installed", packageName, installed)
		return nil
	}

	architecture, err := updater.runner.Output(commandContext, "dpkg", "--print-architecture")
	if err != nil {
		return err
	}
	architecture = strings.TrimSpace(architecture)
	metadataOutput, err := updater.runner.Output(commandContext, "apt-cache", "show", "--no-all-versions", packageName+"="+candidate)
	if err != nil {
		return err
	}
	metadata, err := parsePackageMetadata(metadataOutput, candidate, architecture)
	if err != nil {
		return err
	}

	uriArguments := append([]string{}, aptSecurityOptions...)
	// apt-get historically defaults --print-uris to MD5 for compatibility.
	// Force the same strong hash that we validated in the signed Packages index.
	uriArguments = append(uriArguments, "-o", "Acquire::ForceHash=sha256", "--print-uris", "download", packageName+"="+candidate)
	uriOutput, err := updater.runner.Output(commandContext, "apt-get", uriArguments...)
	if err != nil {
		return err
	}
	if err := validateDownloadURI(uriOutput, metadata); err != nil {
		return err
	}

	if updater.config.CheckOnly {
		log.Printf("validated update %s -> %s (check-only)", installed, candidate)
		return nil
	}

	wasActive := updater.serviceIsActive(commandContext)
	log.Printf("installing authenticated update %s -> %s", installed, candidate)
	if err := updater.aptGet(commandContext, "install", "--yes", "--only-upgrade", "--no-remove", packageName+"="+candidate); err != nil {
		return fmt.Errorf("could not install %s: %w", candidate, err)
	}

	installedAfter, err := updater.installedVersion(commandContext)
	if err != nil {
		return err
	}
	if installedAfter != candidate {
		return fmt.Errorf("installed version %s does not equal validated candidate %s", installedAfter, candidate)
	}
	if wasActive {
		if err := updater.waitForHealth(ctx, candidate); err != nil {
			return err
		}
	}
	log.Printf("successfully updated %s to %s", packageName, candidate)
	return nil
}

func (updater *Updater) aptGet(ctx context.Context, arguments ...string) error {
	args := append([]string{}, aptSecurityOptions...)
	args = append(args, arguments...)
	return updater.runner.Run(ctx, "apt-get", args...)
}

func (updater *Updater) installedVersion(ctx context.Context) (string, error) {
	version, err := updater.runner.Output(ctx, "dpkg-query", "--show", "--showformat=${Version}", packageName)
	if err != nil {
		return "", fmt.Errorf("could not determine installed %s version: %w", packageName, err)
	}
	version = strings.TrimSpace(version)
	if !debianVersionPattern.MatchString(version) {
		return "", fmt.Errorf("dpkg returned invalid installed version %q", version)
	}
	return version, nil
}

func (updater *Updater) versionIsGreater(ctx context.Context, candidate, installed string) (bool, error) {
	err := updater.runner.Run(ctx, "dpkg", "--compare-versions", candidate, "gt", installed)
	if err == nil {
		return true, nil
	}
	if updater.runner.Run(ctx, "dpkg", "--compare-versions", candidate, "le", installed) == nil {
		return false, nil
	}
	return false, fmt.Errorf("could not compare Debian versions %q and %q", candidate, installed)
}

func (updater *Updater) packageIsHeld(ctx context.Context) (bool, error) {
	heldPackages, err := updater.runner.Output(ctx, "apt-mark", "showhold")
	if err != nil {
		return false, err
	}
	for _, heldPackage := range strings.Fields(heldPackages) {
		if heldPackage == packageName {
			return true, nil
		}
	}
	return false, nil
}

func (updater *Updater) serviceIsActive(ctx context.Context) bool {
	return updater.runner.Run(ctx, "systemctl", "is-active", "--quiet", "mysterium-node.service") == nil
}

func (updater *Updater) waitForHealth(ctx context.Context, candidate string) error {
	healthContext, cancel := context.WithTimeout(ctx, updater.config.HealthTimeout)
	defer cancel()
	expectedVersion := strings.SplitN(candidate, "+", 2)[0]
	expectedVersion = strings.Replace(expectedVersion, "~rc", "-rc", 1)

	ticker := time.NewTicker(updater.config.HealthInterval)
	defer ticker.Stop()
	for {
		if err := updater.checkHealth(healthContext, expectedVersion); err == nil {
			return nil
		}
		select {
		case <-healthContext.Done():
			return fmt.Errorf("mysterium-node did not become healthy at version %s within %s", expectedVersion, updater.config.HealthTimeout)
		case <-ticker.C:
		}
	}
}

func (updater *Updater) checkHealth(ctx context.Context, expectedVersion string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, updater.config.HealthcheckURL, nil)
	if err != nil {
		return err
	}
	response, err := updater.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck returned %s", response.Status)
	}
	var health struct {
		Version string `json:"version"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxHealthResponseBytes))
	if err := decoder.Decode(&health); err != nil {
		return fmt.Errorf("invalid healthcheck response: %w", err)
	}
	if health.Version != expectedVersion {
		return fmt.Errorf("healthcheck reports version %q, expected %q", health.Version, expectedVersion)
	}
	return nil
}
