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
	"flag"
	"testing"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/boltdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestCommandRun(t *testing.T) {
	// given
	tempDir := boltdbtest.CreateTempDir(t)
	defer boltdbtest.RemoveTempDir(t, tempDir)

	cmdOutput := bytes.NewBufferString("")

	// when
	config.FlagDataDir.Value = tempDir
	config.FlagRuntimeDir.Value = tempDir
	cmd := NewCommand()
	err := cmd.Run(cli.NewContext(
		&cli.App{Writer: cmdOutput},
		flag.NewFlagSet("test", 0),
		nil,
	))

	// then
	assert.NoError(t, err)
	assert.Contains(t, cmdOutput.String(), `user password changed successfully`)

	storage, err := boltdb.NewStorage(tempDir)
	assert.NoError(t, err)

	defer func() { _ = storage.Close() }()

	err = auth.NewCredentials(config.FlagTequilapiUsername.Value, config.FlagTequilapiPassword.Value, storage).Validate()
	assert.NoError(t, err)
}
