package cockpit_srv

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/rs/zerolog"
)

// createSPAHandler creates a handler for serving a SPA
// It serves static files if they exist, otherwise falls back to 200.html
func createSPAHandler(webFS fs.FS, log zerolog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path
		requestPath := path.Clean(r.URL.Path)

		// Security: prevent directory traversal
		if strings.Contains(requestPath, "..") {
			log.Warn().Str("path", requestPath).Msg("attempted directory traversal")
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// Try to open the file
		filePath := strings.TrimPrefix(requestPath, "/")
		file, err := webFS.Open(filePath)
		if err == nil {
			// File exists, check if it's a directory
			stat, err := file.Stat()
			file.Close()
			if err == nil && !stat.IsDir() {
				// It's a file, serve it with proper caching headers
				w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year for assets
				http.FileServer(http.FS(webFS)).ServeHTTP(w, r)
				return
			}
		}

		// File doesn't exist or is a directory, serve 200.html for SPA routing.
		// 200.html uses absolute asset paths and a fixed empty base, so it works
		// correctly when served for any URL depth (e.g. /devices/gimli).
		// index.html uses relative paths and a runtime-computed base that breaks
		// for nested routes.
		indexData, err := fs.ReadFile(webFS, "200.html")
		if err != nil {
			log.Error().Err(err).Msg("failed to read 200.html")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // Don't cache 200.html
		w.WriteHeader(http.StatusOK)
		w.Write(indexData)
	})
}
