package health

import (
	"net/http"

	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	isReady bool

	readyStatus = metrics.NewGauge(prometheus.GaugeOpts{
		Subsystem: "health",
		Name:      "ready",
		Help:      "Status of readiness of running process",
	})
	aliveStatus = metrics.NewGauge(prometheus.GaugeOpts{
		Subsystem: "health",
		Name:      "alive",
		Help:      "Status of aliveness of running process",
	})
)

func init() {
	aliveStatus.Set(1)
	readyStatus.Set(0)
}

func IsReady() bool {
	return isReady
}

func SetReady() {
	readyStatus.Set(1)
	isReady = true
}

func SetUnready() {
	readyStatus.Set(0)
	isReady = false
}

func RegisterRoutes(r *http.ServeMux) {
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
