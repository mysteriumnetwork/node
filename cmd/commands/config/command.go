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

package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/urfavecli/clicontext"

	"github.com/urfave/cli/v2"
)

// CommandName is the name which is used to call this command
const CommandName = "config"

// NewCommand function creates license command.
func NewCommand() *cli.Command {
	cmd := &command{}
	return &cli.Command{
		Name:        CommandName,
		Usage:       "Manage your node config",
		Description: "Using config subcommands you can view and manage your current node config",
		Before: func(ctx *cli.Context) error {
			if err := clicontext.LoadUserConfigQuietly(ctx); err != nil {
				return err
			}

			config.ParseFlagsServiceStart(ctx)
			config.ParseFlagsServiceOpenvpn(ctx)
			config.ParseFlagsServiceWireguard(ctx)
			config.ParseFlagsServiceNoop(ctx)
			config.ParseFlagsNode(ctx)
			return nil
		},
		Subcommands: []*cli.Command{
			{
				Name:  "show",
				Usage: "Show current config",
				Action: func(ctx *cli.Context) error {
					cmd.show()
					return nil
				},
			},
		},
	}
}

type command struct{}

func (c *command) show() {
	dest := map[string]string{}
	squishMap(config.Current.GetConfig(), dest)

	if len(dest) == 0 {
		clio.Error("Config is empty or impossible to parse")
		return
	}

	printMapOrdered(dest)
}

// Orders keys alphabetically and prints a given map.
func printMapOrdered(m map[string]string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Println(k+":", m[k])
	}
}

// squishMap squishMap a given `source` map by checking every map
// value and converting it to a string.
// If a map value is another map, it will also be parsed and will gain
// a key which is equal to both map keys joined with a `.` symbol.
func squishMap(source map[string]interface{}, dest map[string]string, prefix ...string) {
	keyPrefix := ""
	if len(prefix) != 0 {
		keyPrefix = strings.Join(prefix, ".")
	}

	for k, v := range source {
		if nm, ok := v.(map[string]interface{}); ok {
			squishMap(nm, dest, k)
		} else {
			if keyPrefix != "" {
				k = keyPrefix + "." + k
			}

			dest[k] = fmt.Sprint(v)
		}
	}
}
