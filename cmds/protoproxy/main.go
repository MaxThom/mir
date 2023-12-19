package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maxthom/mir/libs/api/health"
	"github.com/maxthom/mir/libs/config"
	"github.com/rs/zerolog"
	logger "github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const AppName = "protoproxy"

var (
	flagDebug    bool
	flagFilePath string

	cfg = ProtoProxyConfig{
		DebugLogging: false,
		HttpServer: HttpServer{
			Port: 3000,
		},
	}
	appConfig = config.Empty()
	log       = logger.With().Str("component", AppName).Logger()
)

type (
	ProtoProxyConfig struct {
		DebugLogging bool
		HttpServer   HttpServer
	}

	HttpServer struct {
		Port int
	}
)

func init() {
	// Cli
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", AppName)
		fmt.Fprintf(flag.CommandLine.Output(), "  Listen to NatsIO and process protobuf telemetry\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Args:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "Configuration automatically loaded from:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -/etc/%s/config.yaml\n", AppName)
		fmt.Fprintf(flag.CommandLine.Output(), "  -$HOME/.config/%s/config.yaml, if process not zero\n", AppName)
		fmt.Fprintf(flag.CommandLine.Output(), "Example default config.yaml:\n")
		yamlData, _ := yaml.Marshal(cfg)
		fmt.Fprintf(flag.CommandLine.Output(), strings.ReplaceAll("  "+string(yamlData), "\n", "\n  "))
	}

	flag.BoolVar(&flagDebug, "debug", false, "sets log level to debug")
	flag.StringVar(&flagFilePath, "config", "", "pass an extra config file path")
	flag.Parse()

	// Config
	if flagFilePath != "" {
		appConfig = config.New(AppName,
			config.WithEtcFilePath("config.yaml", config.Yaml, false),
			config.WithXdgConfigHomeFilePath("config.yaml", config.Yaml, true),
			config.WithFilePath(flagFilePath, config.Yaml, false),
			config.WithEnvVars(),
		)
	} else {
		appConfig = config.New(AppName,
			config.WithEtcFilePath("config.yaml", config.Yaml, false),
			config.WithXdgConfigHomeFilePath("config.yaml", config.Yaml, true),
			config.WithEnvVars(),
		)
	}
	err := appConfig.LoadAndUnmarshal(&cfg)

	// Logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if flagDebug || cfg.DebugLogging {
		appConfig.Set("debugLogging", true)
		cfg.DebugLogging = true
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

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
