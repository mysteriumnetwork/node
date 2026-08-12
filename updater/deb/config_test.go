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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFromEnvironment(t *testing.T) {
	t.Setenv("MYST_UPDATER_ENABLED", "false")
	t.Setenv("MYST_UPDATER_COMMAND_TIMEOUT", "10m")
	t.Setenv("MYST_UPDATER_HEALTHCHECK_TIMEOUT", "45s")

	config, err := ConfigFromEnvironment(true)
	require.NoError(t, err)
	assert.False(t, config.Enabled)
	assert.True(t, config.CheckOnly)
	assert.Equal(t, 10*time.Minute, config.CommandTimeout)
	assert.Equal(t, 45*time.Second, config.HealthTimeout)
}

func TestConfigRejectsRemoteHealthcheck(t *testing.T) {
	t.Setenv("MYST_UPDATER_HEALTHCHECK_URL", "https://example.com/healthcheck")
	_, err := ConfigFromEnvironment(false)
	assert.Error(t, err)
}
