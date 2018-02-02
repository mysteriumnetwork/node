package cmd

import (
	"fmt"
	"os"
)

type Killer func() error

// NewApplicationStopper invokes all killers and stops application
func NewApplicationStopper(killers ...Killer) func() {
	return newStopper(os.Exit, killers...)
}

type exitter func(code int)

func newStopper(exit exitter, killers ...Killer) func() {
	return func() {
		stop(exit, killers...)
	}
}

func stop(exit exitter, killers ...Killer) {
	killerFailed := false
	for _, kill := range killers {
		err := kill()
		if err != nil {
			msg := fmt.Sprintf("Error while killing process: %v\n", err.Error())
			fmt.Fprintln(os.Stderr, msg)
			killerFailed = true
		}
	}

	fmt.Println("Good bye")

	if killerFailed {
		exit(1)
		return
	}
	exit(0)
}
