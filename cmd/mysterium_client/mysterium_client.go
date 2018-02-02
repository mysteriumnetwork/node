package main

import (
	"fmt"
	"github.com/mysterium/node/cmd"
	"github.com/mysterium/node/cmd/mysterium_client/cli"
	"github.com/mysterium/node/cmd/mysterium_client/run"
	tequilapi_client "github.com/mysterium/node/tequilapi/client"
	"os"
	"path/filepath"
)

func main() {
	options, err := run.ParseArguments(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmdRun := run.NewCommand(options)

	if err := cmdRun.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	killers := []cmd.Killer{cmdRun.Kill}

	if options.CLI {
		cmdCli := cli.NewCommand(
			filepath.Join(options.DirectoryRuntime, ".cli_history"),
			tequilapi_client.NewClient(options.TequilapiAddress, options.TequilapiPort),
			cmdRun.Kill,
		)
		if err := cmdCli.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		killers = append([]cmd.Killer{cmdCli.Kill}, killers...)
	}

	stopper := cmd.NewApplicationStopper(killers...)
	cmd.NewTerminator(stopper)

	if err = cmdRun.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
