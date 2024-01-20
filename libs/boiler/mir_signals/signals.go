package mir_signals

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

var signalChan chan os.Signal

const (
	PRGSHUTDOWN = CodeSignal(0x20)
)

func Notify(sig ...os.Signal) {
	signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, sig...)
}

func Shutdown() {
	signalChan <- CodeSignal(1)
}

func WaitForOsSignals(shutdownFn func()) {
	for {
		s := <-signalChan
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT:
			log.Info().Msg("received " + s.String() + " signal, shutting down...")
			shutdownFn()
			os.Exit(0)
		case PRGSHUTDOWN:
			log.Info().Msg("received " + s.String() + " signal, shutting down...")
			shutdownFn()
			os.Exit(0)
		default:
			log.Info().Msg("received unknown signal " + s.String())
		}
	}
}

type CodeSignal int

func (s CodeSignal) String() string {
	return "code triggered"
}

func (s CodeSignal) Signal() {
}
