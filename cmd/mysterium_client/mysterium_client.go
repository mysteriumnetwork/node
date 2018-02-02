package main

import (
	"fmt"
	"github.com/mysterium/node/cmd"
	"github.com/mysterium/node/cmd/commands/cli"
	"github.com/mysterium/node/cmd/commands/client"
	tequilapi_client "github.com/mysterium/node/tequilapi/client"
	"os"
	"path/filepath"
)

func main() {

	options, err := client.ParseArguments(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmdRun := client.NewCommand(options)
	cmd.NewTerminator(cmdRun)

	if err := cmdRun.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if options.CLI {
		cmdCli := cli.NewCommand(
			filepath.Join(options.DirectoryRuntime, ".cli_history"),
			tequilapi_client.NewClient(options.TequilapiAddress, options.TequilapiPort),
			newStopHandler(cmdRun),
		)
		cmd.NewTerminator(cmdCli)
		if err := cmdCli.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err = cmdRun.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newStopHandler(cmdRun *client.Command) func() error {
	return func() error {
		if err := cmdRun.Kill(); err != nil {
			return err
		}

		os.Exit(0)
		return nil
	}
}
