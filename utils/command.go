package utils

import (
	"os/exec"
	"strings"
)

func SplitCommand(command string, commandArguments string) *exec.Cmd {
	args := strings.Split(commandArguments, " ")
	var trimmedArgs []string
	for _, arg := range args {
		trimmedArgs = append(trimmedArgs, strings.TrimSpace(arg))
	}
	return exec.Command(command, trimmedArgs...)
}
