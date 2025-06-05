package main

import (
	"fmt"
	"syscall"
	"time"

	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog/log"
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
		Password string `cfg:"secret"`
	}
)

const (
	AppName = "example_module"
)

var (
	defaultCfg = CoreConfig{
		LogLevel: "info",
		HttpServer: HttpServer{
			Port: 3016,
		},
		DataBusServer: DataBusServer{
			Url: "nats://127.0.0.1:4222",
		},
		DatabaseServer: DatabaseSever{
			Url:      "ws://127.0.0.1:8000/rpc",
			User:     "root",
			Password: "root",
		},
	}
)

func main() {
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	// Mir
	m, err := mir.Connect(AppName, defaultCfg.DataBusServer.Url, mir.WithDefaultReconnectOpts()...)
	if err != nil {
		panic(err)
	}
	event(m)
	server(m)
	device(m)

	// Handle shutdown
	fmt.Println(fmt.Sprintf("%s initialized", AppName))
	mir_signals.WaitForOsSignals(func() {
		if err := m.Disconnect(); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown Mir")
		}
	})
}

func device(m *mir.Mir) {
	if err := m.Device().Subscribe(
		m.Device().NewSubject("test", "v1", "send"),
		func(msg *mir.Msg, id string, data []byte) {
			fmt.Println("device:", id, string(data))
		}); err != nil {
		fmt.Println("error subscribing to device data")
	}
}

func server(m *mir.Mir) {
	if err := m.Server().Subscribe(
		m.Server().NewSubject("test", "v1", "fn"),
		func(msg *mir.Msg, clientId string, data []byte) {
			fmt.Println("server:", clientId, string(data))
		}); err != nil {
		fmt.Println("error subscribing to server data")
	}

	if err := m.Server().Publish(
		m.Server().NewSubject("test", "v1", "fn"),
		[]byte("hello")); err != nil {
		fmt.Println("error publishing data")
	}
}

func event(m *mir.Mir) {
	if err := m.Event().DeviceOnline().Subscribe(
		func(msg *mir.Msg, deviceId string, d mir_v1.Device, err error) {
			fmt.Println("device online:", deviceId)
		}); err != nil {
		fmt.Println("error subscribing to online event")
	}

	if err := m.Event().DeviceOffline().Subscribe(
		func(msg *mir.Msg, deviceId string, d mir_v1.Device, err error) {
			fmt.Println("device offline:", deviceId)
		}); err != nil {
		fmt.Println("error subscribing to offline event")
	}
	if err := m.Event().Subscribe(
		func(msg *mir.Msg, id string, d mir_v1.EventSpec, err error) {
			fmt.Println("all event:", id)
		}); err != nil {
		fmt.Println("error subscribing to all event")
	}
	if err := m.Event().SubscribeSubject(
		m.Event().NewSubject("*", "example-module", "v1", "device_pdf"),
		func(msg *mir.Msg, id string, d mir_v1.EventSpec, err error) {
			fmt.Println("specific event:", id)
		}); err != nil {
		fmt.Println("error subscribing to specific event")
	}

	time.Sleep(1 * time.Second)
	if err := m.Event().Publish(
		m.Event().NewSubject("test", "example-module", "v1", "device_pdf"),
		mir_v1.EventSpec{
			Type:    mir_v1.EventTypeNormal,
			Reason:  "example_test",
			Message: "a bigger message",
			Payload: []byte{},
			RelatedObject: mir_v1.NewDevice().WithMeta(mir_v1.Meta{
				Name:      "test",
				Namespace: "example-test",
			}).Object}, nil); err != nil {
		fmt.Println("error publishing event")
	}
}
