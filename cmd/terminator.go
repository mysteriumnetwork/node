package cmd

import (
	"os"
	"os/signal"
	"syscall"
)

// Stopper stops application and performs required cleanup tasks
type Stopper func()

// NewTerminator invokes given stopper on SIGTERM and SIGHUP interrupts
func NewTerminator(stop Stopper) {
	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go waitTerminatorSignals(sigterm, stop)
}

func waitTerminatorSignals(terminator chan os.Signal, stop Stopper) {
	<-terminator
	stop()
}
