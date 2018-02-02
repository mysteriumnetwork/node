package cmd

import (
	"fmt"
	"os"
)

type Killer func() error

// NewStopper stops application by invoking given killer and exiting from application
func NewStopper(kill Killer) func() {
	return func() {
		stop(kill)
	}
}

func stop(kill Killer) {
	err := kill()
	if err != nil {
		msg := fmt.Sprintf("Error while killing process: %v\n", err.Error())
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}

	fmt.Println("Good bye")
	os.Exit(0)
}
