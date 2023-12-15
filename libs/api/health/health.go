package health

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

var (
	isReady bool
)

func IsReady() bool {
	return isReady
}

func SetReady() {
	isReady = true
}

func SetUneady() {
	isReady = false
}

func RegisterRoutes(r *chi.Mux) {
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if IsReady() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not ready"))

		}
	})
	r.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("alive"))
	})
}
