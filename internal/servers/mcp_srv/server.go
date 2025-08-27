package mcp_srv

import (
	"context"

	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
)

type MCPServer struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	m         *mir.Mir
}

const (
	ServiceName = "mir_mcp"
)

var (
	// requestTotal = metrics.NewCounterVec(prometheus.CounterOpts{
	// 	Subsystem: "core",
	// 	Name:      "request_total",
	// 	Help:      "Number of request for core",
	// }, []string{"route"})
	// requestErrorTotal = metrics.NewCounterVec(prometheus.CounterOpts{
	// 	Subsystem: "core",
	// 	Name:      "request_error_total",
	// 	Help:      "Number of error request for core",
	// }, []string{"route"})
	l zerolog.Logger
)

func init() {
	// requestTotal.With(prometheus.Labels{"route": "list"}).Add(0)
	// requestTotal.With(prometheus.Labels{"route": "create"}).Add(0)
	// requestTotal.With(prometheus.Labels{"route": "update"}).Add(0)
	// requestTotal.With(prometheus.Labels{"route": "delete"}).Add(0)
	// requestErrorTotal.With(prometheus.Labels{"route": "list"}).Add(0)
	// requestErrorTotal.With(prometheus.Labels{"route": "create"}).Add(0)
	// requestErrorTotal.With(prometheus.Labels{"route": "update"}).Add(0)
	// requestErrorTotal.With(prometheus.Labels{"route": "delete"}).Add(0)
}

func NewMCP(logger zerolog.Logger, m *mir.Mir) (*MCPServer, error) {
	l = logger.With().Str("srv", ServiceName).Logger()

	ctx, cancelFn := context.WithCancel(context.Background())
	return &MCPServer{
		ctx:       ctx,
		cancelCtx: cancelFn,
		m:         m,
	}, nil
}

func (s *MCPServer) Serve() error {

	return nil
}

func (s *MCPServer) Shutdown() error {
	s.cancelCtx()
	return nil
}
