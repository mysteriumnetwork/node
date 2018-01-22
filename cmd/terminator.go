package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type Command interface {
	Run() error
	Wait() error
	Kill() error
}

func NewTerminator(command Command) {
	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go waitTerminatorSignals(sigterm, command)
}

func waitTerminatorSignals(terminator chan os.Signal, command Command) {
	<-terminator

	err := command.Kill()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to kill one of subroutines %q\n", err.Error())
		os.Exit(1)
	}

	fmt.Println("Good bye")
	os.Exit(0)
}
