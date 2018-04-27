package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"sync"
)

// ApplicationStopper stops application and performs required cleanup tasks
type ApplicationStopper func()

// StopOnInterrupts invokes given stopper on SIGTERM and SIGHUP interrupts with additional wait condition
func StopOnInterruptsConditional(stop ApplicationStopper, stopWaiter *sync.WaitGroup) {
	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go waitTerminationSignalConditional(sigterm, stop, stopWaiter)
}

func waitTerminationSignalConditional(termination chan os.Signal, stop ApplicationStopper, stopWaiter *sync.WaitGroup) {
	stopWaiter.Wait()
	<-termination
	stop()
}

// StopOnInterrupts invokes given stopper on SIGTERM and SIGHUP interrupts
func StopOnInterrupts(stop ApplicationStopper) {
	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go waitTerminationSignal(sigterm, stop)
}

func waitTerminationSignal(termination chan os.Signal, stop ApplicationStopper) {
	<-termination
	stop()
}
