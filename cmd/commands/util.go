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

package commands

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	remote_config "github.com/mysteriumnetwork/node/config/remote"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
)

// InitClientAndConfig - initializes and returns a pointer to tequilapi client - also fetches config using it
func InitClientAndConfig(ctx *cli.Context) (*tequilapi_client.Client, error) {
	address := remote_config.TequilAPIAddress(ctx)
	port := remote_config.TequilAPIPort(ctx)
	client := tequilapi_client.NewClient(address, port)

	err := remote_config.RefreshRemoteConfig(client)
	if err != nil {
		clio.Error("failed to connect to node via url:", address+":"+fmt.Sprint(port))
		return nil, err
	}

	return client, nil
}
