package mcp_srv

import (
	"context"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
)

type MCPServer struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	m         *mir.Mir
	mcp       *server.MCPServer
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
	s.mcp = server.NewMCPServer(
		"Mir 🛰️",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)
	s.getDevicesTool()

}

func (s *MCPServer) RegisterRoutes(r *http.ServeMux) {
	// Not implemented yet
	//server.AddMCPRoutes(r, s.mcp, "/mcp")
}

func (s *MCPServer) GetHttpServer() *server.StreamableHTTPServer {
	return server.NewStreamableHTTPServer(s.mcp)
}

func (s *MCPServer) Serve(addr string) error {
	// httpServer := server.NewStreamableHTTPServer(s.mcp)
	// if err := httpServer.Start(addr); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := server.ServeStdio(s.mcp); err != nil {
	// 	fmt.Printf("Server error: %v\n", err)
	// }
	return nil
}

func (s *MCPServer) Shutdown() error {
	s.cancelCtx()

	return nil
}
