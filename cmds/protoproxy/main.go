package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maxthom/mir/libs/api/health"
	"github.com/maxthom/mir/libs/boiler/cli"
	"github.com/maxthom/mir/libs/boiler/config"
	"github.com/rs/zerolog"
	logger "github.com/rs/zerolog/log"
)

const AppName = "protoproxy"

var (
	flagDebug    bool
	flagFilePath string
	flagLogLevel string
	flagManual   bool

	cfg = ProtoProxyConfig{
		LogLevel: "info",
		HttpServer: HttpServer{
			Port: 3000,
		},
	}
	appConfig = config.Empty()
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

func init() {
	mirCli := cli.New(AppName,
		cli.WithDescription("Listen to NatsIO, deserialize protofbuf and push to puthost"),
		cli.WithConfigFilePath(&flagFilePath),
		cli.WithLogLevel(&flagLogLevel),
		cli.WithLogDebug(&flagDebug),
		cli.WithManual(&flagManual,
			"Listen to queues from NatsIO and receive protobuf encoding to deserialize at runtime\n"+
				"using an uploaded protobuf definition.The decoded data is pushed to the puthost protocol.",
			&cfg, true, ""),
	)
	flag.Parse()
	if flagManual {
		fmt.Println(mirCli.Manual)
		os.Exit(0)
	}

	// Config
	opts := []func(*config.MirConfig){
		config.WithEtcFilePath("config.yaml", config.Yaml, false),
		config.WithXdgConfigHomeFilePath("config.yaml", config.Yaml, true),
		config.WithEnvVars(),
	}
	if flagFilePath != "" {
		opts = append(opts, config.WithFilePath(flagFilePath, config.Yaml, false))

	}
	appConfig = config.New(AppName, opts...)
	err := appConfig.LoadAndUnmarshal(&cfg)

	// Logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if flagLogLevel != "" {
		cfg.LogLevel = flagLogLevel
	}
	if flagDebug {
		cfg.LogLevel = "debug"
	}
	switch cfg.LogLevel {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		cfg.LogLevel = "info"

	}
	appConfig.Set("logLevel", cfg.LogLevel)

	if err != nil {
		log.Err(err).Msg("")
		os.Exit(1)
	}
	log.Info().Msg(fmt.Sprintf("%s initializing...", AppName))
	log.Info().Str("config", fmt.Sprintf("%v", appConfig.All())).Msg("config loaded")
}

// TODO os signals catch
// TODO logger builder
// TODO make appConfigSetup public and tweak loading
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
