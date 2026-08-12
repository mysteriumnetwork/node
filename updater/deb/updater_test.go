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
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	installed          string
	candidate          string
	held               bool
	trusted            bool
	installedCandidate bool
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func (runner *fakeRunner) Output(_ context.Context, name string, args ...string) (string, error) {
	joined := strings.Join(args, " ")
	switch {
	case name == "apt-cache" && joined == "policy":
		origin := RepositoryOrigin
		if !runner.trusted {
			origin = "untrusted-origin"
		}
		return "Package files:\n 500 https://ppa.launchpadcontent.net/mysteriumnetwork/node/ubuntu jammy/main amd64 Packages\n     release o=" + origin + ",a=jammy,n=jammy,c=main,b=amd64\n", nil
	case name == "apt-cache" && joined == "policy "+PackageName:
		return fmt.Sprintf("%s:\n  Installed: %s\n  Candidate: %s\n  Version table:\n     %s 500\n        500 https://ppa.launchpadcontent.net/mysteriumnetwork/node/ubuntu jammy/main amd64 Packages\n *** %s 100\n        100 /var/lib/dpkg/status\n", PackageName, runner.installed, runner.candidate, runner.candidate, runner.installed), nil
	case name == "dpkg-query":
		if runner.installedCandidate {
			return runner.candidate, nil
		}
		return runner.installed, nil
	case name == "apt-mark":
		if runner.held {
			return PackageName + "\n", nil
		}
		return "", nil
	case name == "dpkg" && joined == "--print-architecture":
		return "amd64\n", nil
	case name == "apt-cache" && strings.HasPrefix(joined, "show --no-all-versions"):
		return fmt.Sprintf("Package: %s\nArchitecture: amd64\nVersion: %s\nFilename: pool/main/m/myst/myst_%s_amd64.deb\nSHA256: %s\n", PackageName, runner.candidate, runner.candidate, testSHA256), nil
	case name == "apt-get" && strings.Contains(joined, "Acquire::ForceHash=sha256 --print-uris download"):
		return fmt.Sprintf("'https://ppa.launchpadcontent.net/mysteriumnetwork/node/ubuntu/pool/main/m/myst/myst_%s_amd64.deb' myst.deb 123 SHA256:%s\n", runner.candidate, testSHA256), nil
	default:
		return "", fmt.Errorf("unexpected output command: %s %s", name, joined)
	}
}

func (runner *fakeRunner) Run(_ context.Context, name string, args ...string) error {
	joined := strings.Join(args, " ")
	switch {
	case name == "apt-get" && strings.HasSuffix(joined, " update"):
		return nil
	case name == "dpkg" && len(args) == 4 && args[0] == "--compare-versions":
		if args[2] == "gt" && runner.candidate != runner.installed {
			return nil
		}
		if args[2] == "le" && runner.candidate == runner.installed {
			return nil
		}
		return fmt.Errorf("comparison is false")
	case name == "systemctl" && joined == "is-active --quiet mysterium-node.service":
		return nil
	case name == "apt-get" && strings.Contains(joined, " install "):
		runner.installedCandidate = true
		return nil
	default:
		return fmt.Errorf("unexpected run command: %s %s", name, joined)
	}
}

func testUpdater(runner *fakeRunner) *Updater {
	return &Updater{
		config: Config{
			Enabled:        true,
			CheckOnly:      true,
			HealthcheckURL: defaultHealthcheckURL,
			CommandTimeout: time.Minute,
			HealthTimeout:  time.Second,
			HealthInterval: time.Millisecond,
		},
		runner: runner,
		client: &http.Client{Timeout: time.Second},
	}
}

func TestUpdaterValidatesUpdateWithoutInstalling(t *testing.T) {
	runner := &fakeRunner{installed: "1.36.4+build1+jammy", candidate: "1.36.5+build2+jammy", trusted: true}
	updater := testUpdater(runner)
	require.NoError(t, updater.Run(context.Background()))
	assert.False(t, runner.installedCandidate)
}

func TestUpdaterRejectsUnexpectedRepositoryOrigin(t *testing.T) {
	runner := &fakeRunner{installed: "1.36.4+build1+jammy", candidate: "1.36.5+build2+jammy", trusted: false}
	updater := testUpdater(runner)
	err := updater.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), RepositoryOrigin)
	assert.False(t, runner.installedCandidate)
}

func TestUpdaterRespectsPackageHold(t *testing.T) {
	runner := &fakeRunner{installed: "1.36.4+build1+jammy", candidate: "1.36.5+build2+jammy", held: true, trusted: true}
	updater := testUpdater(runner)
	require.NoError(t, updater.Run(context.Background()))
	assert.False(t, runner.installedCandidate)
}

func TestUpdaterInstallsAndChecksExactRunningVersion(t *testing.T) {
	runner := &fakeRunner{installed: "1.36.4+build1+jammy", candidate: "1.36.5+build2+jammy", trusted: true}
	updater := testUpdater(runner)
	updater.config.CheckOnly = false
	updater.client = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(`{"version":"1.36.5"}`)),
		}, nil
	})}
	require.NoError(t, updater.Run(context.Background()))
	assert.True(t, runner.installedCandidate)
}
