package main

import (
	"context"
	"io"
	"os"
	"syscall"

	"github.com/maxthom/mir/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/mir_module"
)

// TODO should I add a namespace and devices are stored in namespace
// The namespace could be kind like different tenants or device type
// Means you could add the namespace in the subject
// Cons: Mean the device has to know its namespace

// device.<device_id>.<module>.<function>.<version>
// client.<user_id>.<module>.<function>.<version>

// When not specifying, it will be wildcard

// TODO add reconnection logic accross
// TODO add subject plus function handler to streams
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	s := mir_module.Stream().Device().Core().Hearthbeat().V1Alpha()
	c := mir_module.Stream().Client().Core().Create().V1Alpha()
	b := mir_module.Stream().Device().Core().V1Alpha()
	mir, err := mir_module.Builder().
		//Target("nats://127.0.0.1:4222").
		//LogLevel(mir_module.LogLevelDebug).
		LogWriters([]io.Writer{os.Stdout}).
		DefaultConfigFile(mir_module.Yaml).
		Streams(s, c, b).
		Build()
	if err != nil {
		panic(err)
	}
	//fmt.Println(c.Subject(), s.Subject())
	l := mir.Logger()

	l.Info().Msg("Mir is ready for launch")
	mirWg, err := mir.Launch(ctx)
	if err != nil {
		panic(err)
	}
	l.Info().Msg("Mir is at maxq and nominal")

	mir_signals.WaitForOsSignals(func() {
		cancel()
		mirWg.Wait()
		l.Info().Msg("shutdown")
	})
}
