package servers

// Server defines the interface for a server that can serve requests and shutdown gracefully
type Server interface {
	Serve() error
	Shutdown() error
}
