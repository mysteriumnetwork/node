package main

import (
	"fmt"
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
	if cmdRun.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if options.CLI {
		cmdCli := cli.NewCommand(
			filepath.Join(options.DirectoryRuntime, ".cli_history"),
			tequilapi_client.NewClient(options.TequilapiAddress, options.TequilapiPort),
		)
		err := cmdCli.Run()
		if cmdCli.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err = cmdRun.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
