package cmd

import (
	"os"
	"os/signal"
	"syscall"
)

type Stopper func()

func NewTerminator(stop Stopper) {
	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go waitTerminatorSignals(sigterm, stop)
}

func waitTerminatorSignals(terminator chan os.Signal, stop Stopper) {
	<-terminator
	stop()
}
