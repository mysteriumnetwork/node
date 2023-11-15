/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package reset

import (
	"bytes"
	"os"
	"testing"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/remote"
	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/stretchr/testify/assert"
)

func TestActionRun(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cmdOutput := bytes.NewBufferString("")

	mockusr := "musr"
	mockpass := "mpsw"

	cfg, err := remote.NewConfig(&cfgFether{
		map[string]interface{}{
			config.FlagDataDir.Name:           tempDir,
			config.FlagTequilapiUsername.Name: mockusr,
			config.FlagTequilapiPassword.Name: mockpass,
		},
	})
	assert.NoError(t, err)

	// when
	cmd := resetAction{
		writer: cmdOutput,
		cfg:    cfg,
	}

	t.Run("test reseting tequila credentials", func(t *testing.T) {
		cmd.resetTequilapi()

		// then
		assert.NoError(t, err)
		assert.Contains(t, cmdOutput.String(), `user password changed successfully`)

		config.FlagTequilapiUsername.Value = mockusr
		err = auth.
			NewCredentialsManager(tempDir).
			Validate(mockusr, mockpass)
		assert.NoError(t, err)
	})
}

type cfgFether struct {
	cfg map[string]interface{}
}

func (c *cfgFether) FetchConfig() (map[string]interface{}, error) {
	cfg := config.NewConfig()
	for k, v := range c.cfg {
		cfg.SetDefault(k, v)
	}

	return cfg.GetConfig(), nil
}
