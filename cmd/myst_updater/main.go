/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	debupdater "github.com/mysteriumnetwork/node/updater/deb"
)

func main() {
	checkOnly := flag.Bool("check-only", false, "validate the available update without installing it")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "myst-updater does not accept positional arguments")
		os.Exit(2)
	}
	if os.Geteuid() != 0 {
		log.Fatal("myst-updater must run as root")
	}

	config, err := debupdater.ConfigFromEnvironment(*checkOnly)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := debupdater.New(config).Run(ctx); err != nil {
		log.Fatal(err)
	}
}
