package mcp_srv

import (
	"context"
	"net/http"

	"github.com/maxthom/mir/pkgs/module/mir"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
)

type MCPServer struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	m         *mir.Mir
	mcp       *mcp.Server
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
	srv := &MCPServer{
		ctx:       ctx,
		cancelCtx: cancelFn,
		m:         m,
	}
	srv.initMCPServer()

	return srv, nil
}

func (s *MCPServer) initMCPServer() {
	s.mcp = mcp.NewServer(&mcp.Implementation{Name: "Mir 🛰️", Version: "v1.0.0"}, &mcp.ServerOptions{
		HasPrompts:   true,
		HasResources: true,
		HasTools:     true,
	})

	NewGetDevicesTool(s.m).RegisterTool(s.mcp)
}

func (s *MCPServer) Serve(addr string) error {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return s.mcp
	}, &mcp.StreamableHTTPOptions{
		Stateless: true,
	})
	return http.ListenAndServe(addr, handler)
}

func (s *MCPServer) Shutdown() error {
	s.cancelCtx()

	return nil
}
