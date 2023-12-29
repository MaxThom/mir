package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/protoproxy/protoproxyconnect"
	"github.com/maxthom/mir/libs/api/health"
	"github.com/maxthom/mir/libs/api/metrics"
	"github.com/maxthom/mir/libs/boiler/mir_cli"
	"github.com/maxthom/mir/libs/boiler/mir_config"
	"github.com/maxthom/mir/libs/boiler/mir_log"
	"github.com/maxthom/mir/libs/boiler/mir_signals"
	proto_store "github.com/maxthom/mir/libs/proto/store"
	"github.com/maxthom/mir/services/protoproxy"
	logger "github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const AppName = "protoproxy"
const Version = "0.0.1"

var (
	flagDebug       bool
	flagFilePath    string
	flagLogLevel    string
	flagSchemaPaths []string

	cfg = ProtoProxyConfig{
		LogLevel: "info",
		HttpServer: HttpServer{
			Port: 3000,
		},
	}
	appConfig = mir_config.Empty()
	log       = logger.With().Str("component", AppName).Logger()
)

type (
	ProtoProxyConfig struct {
		LogLevel   string
		HttpServer HttpServer
	}

	HttpServer struct {
		Port int
	}
)

// TODO
//   - Library to translate proto to influx
//   - The api could have an endpoint to send telemetry as json
//   - The api could have an endpoint to send telemetry as grpc with a dynamic grpc server
func main() {
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)
	mux := http.NewServeMux()
	api := http.NewServeMux()

	// Services
	// ProtoProxy
	pp := &protoproxy.ProtoProxyServer{}
	protoproxy.RegisterMetrics(metrics.Registry())
	api.Handle(protoproxyconnect.NewProtoProxyServiceHandler(pp))

	// ProtoStore
	// Args are all proto binary to load as path
	for _, p := range mir_cli.Args() {
		err := proto_store.GlobalRegistry.LoadProtoBinaryFileFromDisk(proto_store.Meta{
			Name: "todo",
			Desc: "a description",
			Tags: map[string]string{"ca": "dev"},
		}, p)
		if err != nil {
			logger.Err(err).Msg("")
		}
	}
	// TODO protostore service take a registry in the constructor
	//      in the future, this could be an interface to many store type

	// Metrics & Health
	metrics.RegisterRoutes(mux)
	health.RegisterRoutes(mux)
	health.SetReady()

	// Launch server
	mux.Handle("/api/", http.StripPrefix("/api", api))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HttpServer.Port),
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Err(err).Msg("")
		}
	}()

	// Handle shutdown
	log.Info().Msg(fmt.Sprintf("%s initialized", AppName))
	mir_signals.WaitForOsSignals(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to gracefully shutdown server")
		}
	})
}

func init() {
	// Cli
	mir_cli.Setup(AppName,
		mir_cli.WithDescription("Listen to NatsIO, deserialize protofbuf and push to puthost"),
		mir_cli.WithConfigFilePath(&flagFilePath),
		mir_cli.WithLogLevel(&flagLogLevel),
		mir_cli.WithLogDebug(&flagDebug),
		mir_cli.WithManual(
			"Listen to queues from NatsIO and receive protobuf encoding to deserialize at runtime\n"+
				"using an uploaded protobuf definition.The decoded data is pushed to the puthost protocol.",
			&cfg, true, ""),
		mir_cli.WithOsFlag(func() {
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
	err, warns := appConfig.LoadAndUnmarshal(&cfg)

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
	if warns != nil {
		log.Warn().Msg(warns.Error())
	}

	log.Info().Msg(fmt.Sprintf("%s initializing...", AppName))
	log.Info().Str("mir_config", fmt.Sprintf("%v", appConfig.All())).Msg("mir_config loaded")
}
