// Package pprof provides HTTP handlers for runtime profiling data.
// It wraps the standard net/http/pprof package handlers and provides
// a convenient RegisterRoutes function to register all pprof endpoints.
package pprof

import (
	"net/http"
	"net/http/pprof"
	"os"
)

// RegisterRoutes registers all pprof HTTP handlers on the provided mux.
// This includes:
//   - /debug/pprof/           - Shows index page with list of available profiles
//   - /debug/pprof/cmdline    - Shows the command line invocation of the current program
//   - /debug/pprof/profile    - CPU profile (default 30s, configurable via ?seconds=N)
//   - /debug/pprof/symbol     - Symbol lookup for program counters
//   - /debug/pprof/trace      - Execution trace (default 1s, configurable via ?seconds=N)
//
// And runtime profiles accessible via /debug/pprof/{profile}:
//   - goroutine    - Stack traces of all current goroutines
//   - heap         - Heap memory allocations sampling
//   - allocs       - All past memory allocations sampling
//   - threadcreate - Stack traces that led to thread creation
//   - block        - Stack traces that led to blocking on synchronization primitives
//   - mutex        - Stack traces of holders of contended mutexes
//
// Security Note: These endpoints expose sensitive debugging information and
// should only be accessible in development or on a secured internal port.
func RegisterRoutes(mux *http.ServeMux) {
	// Register the index handler
	mux.HandleFunc("/debug/pprof/", pprof.Index)

	// Register specific endpoint handlers
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Note: The runtime profiles (goroutine, heap, allocs, threadcreate, block, mutex)
	// are automatically handled by pprof.Index and don't need separate registration.
	// They are accessible via /debug/pprof/{profile_name}
}

// RegisterRoutesIfEnvGoPprofSet conditionally registers all pprof HTTP handlers
// on the provided mux only if the GO_PPROF environment variable is set to a
// non-empty value. This provides a convenient way to enable profiling endpoints
// in production environments without code changes.
//
// Usage:
//   - Set GO_PPROF=1 (or any non-empty value) to enable profiling endpoints
//   - Leave GO_PPROF unset or empty to disable profiling endpoints
//
// This is useful for production deployments where you want the ability to
// enable profiling on demand without redeploying or modifying code.
//
// Security Note: The same security considerations apply as with RegisterRoutes.
// Ensure proper access controls are in place when enabling in production.
func RegisterRoutesIfEnvGoPprofSet(mux *http.ServeMux) {
	if os.Getenv("GO_PPROF") == "" {
		return
	}

	RegisterRoutes(mux)
}
