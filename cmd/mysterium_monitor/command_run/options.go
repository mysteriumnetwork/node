package command_run

import (
	"bufio"
	"errors"
	"flag"
	"os"
)

type CommandOptions struct {
	DirectoryRuntime string
	NodeKeys         []string
}

func ParseArguments(args []string) (options CommandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.StringVar(
		&options.DirectoryRuntime,
		"runtime-dir",
		".",
		"Runtime directory for temp files (should be writable)",
	)

	var nodeToCheck string
	flags.StringVar(
		&nodeToCheck,
		"node",
		"",
		"Node to be checked",
	)

	var fileToCheck string
	flags.StringVar(
		&fileToCheck,
		"node-file",
		"",
		"File with node list to be checked",
	)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	if nodeToCheck != "" {
		options.NodeKeys = []string{nodeToCheck}
	} else if fileToCheck != "" {
		options.NodeKeys, err = parseLines(fileToCheck)
	} else {
		err = errors.New("Provide which nodes to monitor!")
	}

	return options, err
}

func parseLines(filePath string) (lines []string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	file.Close()
	return
}
