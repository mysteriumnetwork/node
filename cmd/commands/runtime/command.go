/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

// Package runtime exposes host runtime diagnostics without starting the node.
package runtime

import (
	"encoding/json"

	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/config"
	runtime_service "github.com/mysteriumnetwork/node/services/runtime/service"
	runtime_capabilities "github.com/mysteriumnetwork/runtime/capabilities"
)

// CommandName is the name used to invoke runtime diagnostics.
const CommandName = "runtime"

type backendFactory func(dataDir string) runtime_service.Backend

type statusOutput struct {
	Capabilities runtime_capabilities.RuntimeCapabilities  `json:"capabilities"`
	Detailed     runtime_capabilities.DetailedCapabilities `json:"detailed"`
	Runtime      runtime_service.RuntimeStatus             `json:"runtime"`
}

// NewCommand creates the runtime diagnostics command.
func NewCommand() *cli.Command {
	return newCommand(runtime_service.NewBackend)
}

func newCommand(newBackend backendFactory) *cli.Command {
	backend := func(ctx *cli.Context) runtime_service.Backend {
		return newBackend(ctx.String(config.FlagDataDir.Name))
	}

	return &cli.Command{
		Name:        CommandName,
		Usage:       "Inspect workload runtime support",
		Description: "Runtime diagnostics probe the current host without starting the node or publishing a service proposal.",
		Flags:       []cli.Flag{&config.FlagDataDir},
		Subcommands: []*cli.Command{
			{
				Name:  "status",
				Usage: "Output runtime capabilities and overall isolation status as JSON",
				Action: func(ctx *cli.Context) error {
					runtimeBackend := backend(ctx)
					simple, detailed := runtimeBackend.Capabilities()
					return writeJSON(ctx, statusOutput{
						Capabilities: simple,
						Detailed:     detailed,
						Runtime:      runtimeBackend.Status(),
					})
				},
			},
		},
	}
}

func writeJSON(ctx *cli.Context, value interface{}) error {
	encoder := json.NewEncoder(ctx.App.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
