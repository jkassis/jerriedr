package kittie

import (
	context "context"
	"net"
	"os"
	"os/signal"
	"strconv"

	"github.com/jkassis/jerrie/core"
)

// Handler represents an function that reads from a reader and writes to a writer
// returns the res passed in and and error
// the res is returned for tests but typically ignored by other code
type Handler func(ctx context.Context, req []byte) (res []byte, err error)

// ServiceShutdownGracePeriodGet returns the grace periof for the service when a shutdown
// signal is received. Set from env var... SERVICE_SHUTDOWN_GRACE_PERIOD
func ServiceShutdownGracePeriodGet() int {
	// Set shutdown grace period
	serviceShutdownGracePeriod := os.Getenv("SERVICE_SHUTDOWN_GRACE_PERIOD")
	shutdownGracePeriod, err := strconv.Atoi(serviceShutdownGracePeriod)
	if err != nil {
		core.Log.Error("SHUTDOWN_GRACE_PERIOD Invalid")
		os.Exit(1)
	}
	// core.Log.Debug("SERVICE_SHUTDOWN_GRACE_PERIOD=" + serviceShutdownGracePeriod)
	return shutdownGracePeriod
}

// ServiceWaitForOSSignal Waits for an os signal
func ServiceWaitForOSSignal() os.Signal {
	// Create a channel for os signals and block
	signalChannel := make(chan os.Signal, 1)
	defer close(signalChannel)

	signal.Notify(signalChannel, os.Kill, os.Interrupt)
	for {
		incomingSignal := <-signalChannel
		core.Log.Warn("Got os.Signal : " + incomingSignal.String())
		if incomingSignal == os.Kill || incomingSignal == os.Interrupt {
			return incomingSignal
		}
	}
}

// HostportDelta returns a new hostport with the port delta applied
func HostportDelta(hostport string, delta int) (string, error) {
	var err error
	if host, port, err := net.SplitHostPort(string(hostport)); err == nil {
		if porti, err := strconv.Atoi(port); err == nil {
			return host + ":" + strconv.Itoa(porti+delta), nil
		}
	}
	return "", err
}
