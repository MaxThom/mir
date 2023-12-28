package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/protoproxy"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/protoproxy/protoproxyconnect"
	"github.com/maxthom/mir/libs/api/health"
	"github.com/maxthom/mir/libs/api/metrics"
	"github.com/maxthom/mir/libs/boiler/mir_cli"
	"github.com/maxthom/mir/libs/boiler/mir_config"
	"github.com/maxthom/mir/libs/boiler/mir_log"
	"github.com/maxthom/mir/libs/boiler/mir_signals"
	"github.com/prometheus/client_golang/prometheus"
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
	appConfig                             = mir_config.Empty()
	log                                   = logger.With().Str("component", AppName).Logger()
	uploadSchemaMetric prometheus.Counter = nil
)

type (
	ProtoProxyConfig struct {
		LogLevel   string
		HttpServer HttpServer
	}

	HttpServer struct {
		Port int
	}

	ProtoProxyServer struct{}
)

func (p *ProtoProxyServer) UploadSchema(ctx context.Context,
	req *connect.Request[protoproxy.UploadSchemaRequest],
) (*connect.Response[protoproxy.UploadSchemaResponse], error) {
	uploadSchemaMetric.Inc()
	log.Info().Msg("upload schema!")
	log.Info().Msg(fmt.Sprintf("Request headers: %s", req.Header()))
	res := connect.NewResponse(&protoproxy.UploadSchemaResponse{
		Msg: "schema uploaded!",
	})
	res.Header().Set("Content-Type", "application/json")
	return res, nil
}

func main() {
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	mux := http.NewServeMux()

	// Services
	// Connect
	api := http.NewServeMux()
	pp := &ProtoProxyServer{}
	api.Handle(protoproxyconnect.NewProtoProxyServiceHandler(pp))
	mux.Handle("/api/", http.StripPrefix("/api", api))

	// Metrics & Health
	metrics.RegisterMirMetrics(AppName, Version, map[string]string{"ca": "dev"}, fmt.Sprintf("%v", appConfig.All()))
	uploadSchemaMetric = metrics.NewCounter(prometheus.CounterOpts{
		Name: "upload_schema_counter",
		Help: "Upload schema",
	})
	metrics.Register(uploadSchemaMetric)
	metrics.RegisterRoutes(mux)
	health.RegisterRoutes(mux)
	health.SetReady()

	// Launch server
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
