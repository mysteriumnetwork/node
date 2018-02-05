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

	var cmdRun *client.Command
	var cmdCli *cli.Command

	stop := func() {
		var killers []cmd.Killer
		if cmdRun != nil {
			killers = append(killers, cmdRun.Kill)
		}
		// TODO: add CLI killer once it's not blocking anymore
		stop := cmd.NewApplicationStopper(killers...)
		stop()
	}

	cmdRun = client.NewCommand(options, stop)

	if err := cmdRun.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if options.CLI {
		cmdCli = cli.NewCommand(
			filepath.Join(options.DirectoryRuntime, ".cli_history"),
			tequilapi_client.NewClient(options.TequilapiAddress, options.TequilapiPort),
			stop,
		)
		if err := cmdCli.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	cmd.NewTerminator(stop)

	if err = cmdRun.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
