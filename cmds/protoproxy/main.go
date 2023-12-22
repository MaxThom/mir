package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maxthom/mir/libs/api/health"
	"github.com/maxthom/mir/libs/boiler/mir_cli"
	"github.com/maxthom/mir/libs/boiler/mir_config"
	"github.com/maxthom/mir/libs/boiler/mir_log"
	logger "github.com/rs/zerolog/log"
)

const AppName = "protoproxy"

var (
	flagDebug    bool
	flagFilePath string
	flagLogLevel string

	cfg = ProtoProxymir_config{
		LogLevel: "info",
		HttpServer: HttpServer{
			Port: 3000,
		},
	}
	appConfig = mir_config.Empty()
	log       = logger.With().Str("component", AppName).Logger()
)

type (
	ProtoProxymir_config struct {
		LogLevel   string
		HttpServer HttpServer
	}

	HttpServer struct {
		Port int
	}
)

// TODO os signals catch
// TODO logger builder
// TODO make appmir_configSetup public and tweak loading
func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("%v", health.IsReady())))
	})

	health.RegisterRoutes(r)
	health.SetReady()
	log.Info().Msg(fmt.Sprintf("%s initialized", AppName))
	http.ListenAndServe(fmt.Sprintf(":%d", cfg.HttpServer.Port), r)
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
