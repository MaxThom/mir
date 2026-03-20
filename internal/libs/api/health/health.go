package health

import (
	"net/http"
	"strings"
	"sync"

	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type ComponentStatus string

const (
	ComponentStatusReady    ComponentStatus = "ready"
	ComponentStatusDegraded ComponentStatus = "degraded"
	ComponentStatusUnready  ComponentStatus = "unready"

	ComponentSurreal string = "surreal"
	ComponentInflux  string = "influx"
	ComponentNats    string = "nats"
	ComponentHttp    string = "http"
)

var (
	componentStatus = make(map[string]ComponentStatus)
	componentLock   = sync.RWMutex{}
	isReady         = ComponentStatusUnready

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
	compReadyStatus = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "health",
		Name:      "component_ready",
		Help:      "Status of readiness of each component",
	}, []string{"component"})
)

func init() {
	aliveStatus.Set(1)
	readyStatus.Set(0)
}

func SetReady() {
	isReady = ComponentStatusReady
	readyStatus.Set(1)
}

func SetUnready() {
	isReady = ComponentStatusUnready
	readyStatus.Set(0)
}

func SetDegraded() {
	isReady = ComponentStatusDegraded
	readyStatus.Set(0)
}

func IsReady() bool {
	return isReady == ComponentStatusReady
}

func GetReadyStatus() ComponentStatus {
	return isReady
}

func GetComponentStatus(component string) ComponentStatus {
	componentLock.RLock()
	defer componentLock.RUnlock()
	return componentStatus[component]
}

func SetComponentReady(component string) {
	componentLock.Lock()
	defer componentLock.Unlock()
	componentStatus[component] = ComponentStatusReady
	compReadyStatus.With(prometheus.Labels{"component": component}).Set(1)
	for _, status := range componentStatus {
		if status != ComponentStatusReady {
			SetDegraded()
			return
		}
	}
	SetReady()
}

func SetComponentUnready(component string) {
	componentLock.Lock()
	defer componentLock.Unlock()
	componentStatus[component] = ComponentStatusUnready
	compReadyStatus.With(prometheus.Labels{"component": component}).Set(0)
	SetDegraded()
}

func RegisterRoutes(r *http.ServeMux) {
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		var sb strings.Builder
		sb.WriteString(string(isReady) + "\n\nsubsystems:\n")
		for comp, status := range componentStatus {
			sb.WriteString(" " + comp + ": " + string(status) + "\n")
		}
		if GetReadyStatus() == ComponentStatusReady {
			w.WriteHeader(http.StatusOK)
		} else if GetReadyStatus() == ComponentStatusDegraded {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		w.Write([]byte(sb.String()))
	})
	r.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("alive"))
	})
}
