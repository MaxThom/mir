package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maxthom/mir/libs/api/health"
	"github.com/maxthom/mir/libs/config"
)

const AppName = "protoproxy"

var appConfig = config.New(AppName,
	config.WithEtcFilePath("config.yaml", config.Yaml, false),
	config.WithXdgConfigHomeFilePath("config.yaml", config.Yaml, true),
	config.WithEnvVars(),
)

type ProtoProxyConfig struct {
	DebugLogging bool
	HttpServer   HttpServer
}

type HttpServer struct {
	Port int
}

func main() {
	var cfg ProtoProxyConfig
	if err := appConfig.LoadAndUnmarshal(&cfg); err != nil {
		fmt.Println(err)
	}

	// Use the configuration.
	fmt.Printf("config: %v\n", appConfig.All())
	fmt.Printf("cfg: %v\n", cfg)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("%v", health.IsReady())))
	})

	health.RegisterRoutes(r)
	health.SetReady()

	http.ListenAndServe(fmt.Sprintf(":%d", cfg.HttpServer.Port), r)
}
