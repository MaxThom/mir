package mir_signals

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

var signalChan chan os.Signal

func Notify(sig ...os.Signal) {
	signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, sig...)
}

func WaitForOsSignals(shutdownFn func()) {
	for {
		s := <-signalChan
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT:
			log.Info().Msg("received " + s.String() + " signal, shutting down...")
			shutdownFn()
			os.Exit(0)
		default:
			log.Info().Msg("received unknown signal " + s.String())
		}
	}
}
