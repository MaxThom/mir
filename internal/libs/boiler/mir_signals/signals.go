package mir_signals

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

var signalChan chan os.Signal
var sigCtx context.Context
var cancel context.CancelFunc

const (
	PRGSHUTDOWN = CodeSignal(0x20)
)

func NotifyContext(ctx context.Context, sig ...os.Signal) (context.Context, context.CancelFunc) {
	signalChan = make(chan os.Signal, 1)
	sigCtx, cancel = signal.NotifyContext(ctx, sig...)
	return sigCtx, cancel
}

func Notify(sig ...os.Signal) {
	sigCtx, cancel = context.WithCancel(context.Background())
	signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, sig...)
}

func Shutdown() {
	cancel()
	signalChan <- CodeSignal(0x20)
}

func WaitForOsSignalsContext(ctx, shutdownFn func()) {
	select {}
}

// IDEA add context that can be cancelled ? or the cancel func
func WaitForOsSignals(shutdownFn func()) {
	select {
	case <-sigCtx.Done():
		log.Info().Msg("received interrupt signal, shutting down...")
		shutdownFn()
		return
	case <-signalChan:
		s := <-signalChan
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT:
			log.Info().Msg("received " + s.String() + " signal, shutting down...")
			shutdownFn()
			return
		case PRGSHUTDOWN:
			log.Info().Msg("received " + s.String() + " signal, shutting down...")
			shutdownFn()
			return
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
