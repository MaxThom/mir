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
	bus "github.com/maxthom/mir/libs/external/natsio"
	proto_store "github.com/maxthom/mir/libs/proto/store"
	"github.com/maxthom/mir/services/protoproxy"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	logger "github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	AppName = "protoproxy"
	Version = "0.0.1"
)

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
		DataBusServer: DataBusServer{
			Url: "nats://127.0.0.1:4222",
		},
	}
	appConfig = mir_config.Empty()
	log       = logger.With().Str("component", AppName).Logger()
)

type (
	ProtoProxyConfig struct {
		LogLevel      string
		HttpServer    HttpServer
		DataBusServer DataBusServer
	}

	HttpServer struct {
		Port int
	}

	DataBusServer struct {
		Url string
	}
)

// TODO
//   - [x] Library to translate proto to influx
//   - [x] The api could have an endpoint to send telemetry as json
//   - [x] The api could have an endpoint to send telemetry as grpc with a dynamic grpc server
//   - [x] Define route mechanism
//   - [x] Setup Nats
//   - [ ] Setup QuestDB
//   - [ ] Pipe telemetry from Nats to QuestDB using protobuf to line protocol
//   - [ ] Create server side Library
//   - [ ] Create client side Library
//   - [ ] Create the swarm
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

	// Connect to a server
	b, err := bus.New(cfg.DataBusServer.Url,
		bus.WithReconnHandler(func(nc *nats.Conn) {
			logger.Warn().Msg("reconnected to " + nc.ConnectedUrl())
		}),
		bus.WithDisconnHandler(func(nc *nats.Conn, err error) {
			logger.Warn().Msg(fmt.Sprintf("disconnected due to %v, will attempt to reconnect ", err))
		}),
		bus.WithClosedHandler(func(nc *nats.Conn) {
			logger.Warn().Msg("connection to %v closed " + nc.ConnectedUrl())
		}))
	if err != nil {
		logger.Err(err).Msg("")
	}
	logger.Info().Msg("connected to msg bus on " + cfg.DataBusServer.Url)

	// TODO
	// - [ ] worker group on the queue
	js, _ := jetstream.New(b.Conn)
	ctxNats, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := js.CreateOrUpdateStream(ctxNats, jetstream.StreamConfig{
		Name:     bus.TelemetryStreamName,
		Subjects: []string{bus.TelemetryStreamSubject},
	})
	if err != nil {
		fmt.Println(err)
	}

	// retrieve consumer handle from a stream
	cons, err := stream.CreateOrUpdateConsumer(ctxNats, jetstream.ConsumerConfig{
		Durable:        "protoproxy",                         // + hash of pod for scaling?
		FilterSubjects: []string{bus.TelemetryConsumerProto}, // can filter on specific functions
		// Implicit for telemerty, explicity for commands and telemetry
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		fmt.Println(err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		// TODO
		// deserialize from protobuf to line protocol
		// send to database
		fmt.Println("Received jetstream message: ", string(msg.Data()))
		b.Publish(msg.Reply(), []byte("im back!"))
		msg.Ack()
	})
	if err != nil {
		fmt.Println(err)
	}

	defer cc.Stop()

	// Simple Async Subscriber
	// b.Subscribe(">", func(m *nats.Msg) {
	// 	fmt.Printf("Received a message: %s\n", string(m.Data))
	// 	b.Publish(m.Reply, []byte("I can help!"))
	// 	// Header
	// 	// __msg: <proto_message> to id the message
	// 	// __id: <device_id> to id the device
	// 	// the topic will be <device_id>.<type>.<version>
	// 	// eg. ff123b9.tlm.v1alpha
	// 	// crazy idea, the proto_message could be the key.
	// 	// m.Header.Get(key string)
	// })
	//
	// Metrics & Health
	metrics.RegisterRoutes(mux)
	health.RegisterRoutes(mux)

	// Launch server
	mux.Handle("/api/", http.StripPrefix("/api", api))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HttpServer.Port),
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Err(err).Msg("")
			health.SetUneady()
		}
	}()

	// Handle shutdown
	log.Info().Msg(fmt.Sprintf("%s initialized", AppName))
	health.SetReady()
	mir_signals.WaitForOsSignals(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go func() {
			b.Drain()
			b.Close()
		}()

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
