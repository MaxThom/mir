package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/maxthom/mir/libs/api/health"
	"github.com/maxthom/mir/libs/api/metrics"
	"github.com/maxthom/mir/libs/boiler/mir_cli"
	"github.com/maxthom/mir/libs/boiler/mir_config"
	"github.com/maxthom/mir/libs/boiler/mir_log"
	"github.com/maxthom/mir/libs/boiler/mir_signals"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/maxthom/mir/services/core"
	"github.com/nats-io/nats.go"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	AppName = "core"
	Version = "0.0.1"
)

var (
	flagDebug    bool
	flagFilePath string
	flagLogLevel string

	cfg = CoreConfig{
		LogLevel: "info",
		HttpServer: HttpServer{
			Port: 3016,
		},
		DataBusServer: DataBusServer{
			Url: "nats://127.0.0.1:4222",
		},
		DatabaseServer: DatabaseSever{
			Url: "ws://127.0.0.1:8000/rpc",
		},
	}
	appConfig = mir_config.Empty()
	log       = logger.With().Str("cmd", AppName).Logger()
)

type (
	CoreConfig struct {
		LogLevel       string
		HttpServer     HttpServer
		DataBusServer  DataBusServer
		DatabaseServer DatabaseSever
	}

	HttpServer struct {
		Port int
	}

	DataBusServer struct {
		Url string
	}

	DatabaseSever struct {
		Url      string
		User     string
		Password string
	}
)

func run(
	ctx context.Context,
	// args []string,
	// getenv func(string) string,
	// stdin io.Reader,
	// stdout io.Writer,
	// stderr io.Writer,
) error {
	ctx, cancel := context.WithCancel(ctx)
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	mux := http.NewServeMux()

	// Setup
	// Database
	db, err := surrealdb.New(cfg.DatabaseServer.Url)
	if err != nil {
		panic(err)
	}

	if _, err = db.Signin(map[string]any{
		"user": cfg.DatabaseServer.User,
		"pass": cfg.DatabaseServer.Password,
	}); err != nil {
		panic(err)
	}

	if _, err = db.Use("global", "mir"); err != nil {
		panic(err)
	}
	log.Info().Str("url", cfg.DatabaseServer.Url).Str("namespace", "global").Str("database", "mir").Msg("connected to database")

	// Bus
	b, sub, err := subscribeToCoreStream()
	if err != nil {
		log.Err(err).Msg("")
	}
	log.Info().Str("url", cfg.DataBusServer.Url).Str("subject", sub.Subject).Str("queue", sub.Queue).Msg("connected to msg bus")

	// Services
	coreSrv := core.NewCore(log, b, sub, db)
	core.RegisterMetrics(metrics.Registry())

	// Metrics & Health
	metrics.RegisterRoutes(mux)
	health.RegisterRoutes(mux)

	// WebServer
	// api.Handle(protoproxyconnect.NewProtoProxyServiceHandler(pp))
	// mux.Handle("/api/", http.StripPrefix("/api", api))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HttpServer.Port),
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Err(err).Msg("")
			health.SetUneady()
			mir_signals.Shutdown()
		}
	}()

	go func() {
		coreSrv.Listen(ctx)
	}()

	// Handle shutdown
	log.Info().Msg(fmt.Sprintf("%s initialized", AppName))
	health.SetReady()
	mir_signals.WaitForOsSignals(func() {
		cancel()
		go func() {
			b.Drain()
			b.Close()
			db.Close()
		}()

		// 10 secons to close server, gives sometime for bus and puthost
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to gracefully shutdown server")
		}
	})

	return nil
}

// TODO dynamic cli flags based on cfg struct

func main() {
	ctx := context.Background()
	// if err := run(ctx, os.Args, os.Getenv,
	// 	os.Stdin, os.Stdout, os.Stderr); err != nil {
	// 	fmt.Fprintf(os.Stderr, "%s\n", err)
	// 	os.Exit(1)
	// }

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

// TODO
// rework this function to a library or something
func subscribeToCoreStream() (*bus.BusConn, *nats.Subscription, error) {
	b, err := bus.New(cfg.DataBusServer.Url,
		bus.WithReconnHandler(func(nc *nats.Conn) {
			logger.Warn().Msg("reconnected to " + nc.ConnectedUrl())
		}),
		bus.WithDisconnHandler(func(_ *nats.Conn, err error) {
			logger.Warn().Msg(fmt.Sprintf("disconnected due to %v, will attempt to reconnect ", err))
		}),
		bus.WithClosedHandler(func(nc *nats.Conn) {
			logger.Warn().Msg("connection to %v closed " + nc.ConnectedUrl())
		}))
	if err != nil {
		return nil, nil, err
	}

	sub, err := b.SubscribeSync(bus.DeviceStreamSubject)
	if err != nil {
		log.Error().Err(err).Msg("failed to subscribe to subject")
	}

	return b, sub, err
}

func init() {
	// Cli
	mirTarget := ""
	mir_cli.Setup(AppName,
		mir_cli.WithDescription("Listen to NatsIO, manager devices in Mir"),
		mir_cli.WithConfigFilePath(&flagFilePath),
		mir_cli.WithLogLevel(&flagLogLevel),
		mir_cli.WithLogDebug(&flagDebug),
		mir_cli.WithManual(
			"Manager devices for different CRUD operations as well as managing the hearthbeat of devices.",
			&cfg, true, ""),
		mir_cli.WithOsFlag(func() {
			flag.StringVar(&mirTarget, "target", "", "set Mir server url")
		}),
	)
	mir_cli.Parse()

	// Config
	opts := []func(*mir_config.MirConfig){
		mir_config.WithEtcFilePath("config.yaml", mir_config.Yaml, false),
		mir_config.WithXdgConfigHomeFilePath("config.yaml", mir_config.Yaml, true),
		mir_config.WithEnvVars(),
	}
	if flagFilePath != "" {
		opts = append(opts, mir_config.WithFilePath(flagFilePath, mir_config.Yaml, false))
	}
	appConfig = mir_config.New(AppName, opts...)
	err, report := appConfig.LoadAndUnmarshal(&cfg)

	// Logger
	if flagLogLevel != "" {
		cfg.LogLevel = flagLogLevel
	}
	if flagDebug {
		cfg.LogLevel = "debug"
	}
	appConfig.Set("logLevel", cfg.LogLevel)
	mir_log.Setup(mir_log.WithLogLevel(cfg.LogLevel), mir_log.WithTimeFormatUnix())

	// Metrics
	metrics.RegisterMirMetrics(AppName, Version, map[string]string{"ca": "dev"}, fmt.Sprintf("%v", appConfig.All()))

	// Finish
	if err != nil {
		log.Err(err).Msg("")
		os.Exit(1)
	}
	if report != "" {
		log.Info().Msg(report)
	}

	log.Info().Msg(fmt.Sprintf("%s initializing...", AppName))
	log.Info().Str("mir_config", fmt.Sprintf("%v", appConfig.All())).Msg("mir_config loaded")
}
