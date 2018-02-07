package cmd

import (
	"fmt"
	"os"
)

// Killer kills some resource and performs cleanup
type Killer func() error

// NewApplicationStopper invokes killer and stops application
func NewApplicationStopper(kill Killer) func() {
	return newStopper(kill, os.Exit)
}

type exitter func(code int)

func newStopper(kill Killer, exit exitter) func() {
	return func() {
		stop(kill, exit)
	}
}

func stop(kill Killer, exit exitter) {
	if err := kill(); err != nil {
		msg := fmt.Sprintf("Error while killing process: %v\n", err.Error())
		fmt.Fprintln(os.Stderr, msg)
		exit(1)
		return
	}

	fmt.Println("Good bye")
	exit(0)
}
