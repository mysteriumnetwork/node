package cmd

import (
	"os"
	"os/signal"
	"syscall"
)

// Stopper stops application and performs required cleanup tasks
type Stopper func()

// StopOnInterrupts invokes given stopper on SIGTERM and SIGHUP interrupts
func StopOnInterrupts(stop Stopper) {
	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go waitTerminationSignal(sigterm, stop)
}

func waitTerminationSignal(termination chan os.Signal, stop Stopper) {
	<-termination
	stop()
}
