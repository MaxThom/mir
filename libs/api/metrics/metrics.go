// Package metrics provides functionality for registering and managing Prometheus metrics in the Mir system.

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry = prometheus.NewRegistry()

	namespace = "mir"
	subsystem = ""
	extraTags = map[string]string{}
)

// RegisterMirMetrics registers the Mir-specific metrics for the application.
// It takes the following parameters:
//   - appName: The name of the Mir application.
//   - appVersion: The version of the Mir application.
//   - pinnedTags: Additional tags to be added to the metrics.
//   - config: The configuration of the Mir application.
//
// TODO
//   - pinned tag should be use with prometheus scraper instead
//   - perhaps could be use for device id on device lib
func RegisterMirMetrics(appName string, appVersion string, pinnedTags map[string]string, config string) {
	// TODO investigate later how to fill the values
	registry.Register(collectors.NewBuildInfoCollector())
	registry.Register(collectors.NewGoCollector())
	registry.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// set module vars
	subsystem = appName
	extraTags = pinnedTags

	// Mir metrics
	appInfo := NewGauge(prometheus.GaugeOpts{
		Name: "app_info",
		Help: "Mir app info",
		ConstLabels: map[string]string{
			"name":    appName,
			"version": appVersion,
			"config":  config,
		},
	})
	appInfo.Inc()
	registry.Register(appInfo)

	prometheus.DefaultGatherer = registry
	prometheus.DefaultRegisterer = registry
}

// NewCounter creates a new Prometheus Counter metric with the given options.
func NewCounter(opts prometheus.CounterOpts) prometheus.Counter {
	if opts.Namespace == "" {
		opts.Namespace = namespace
	}
	if opts.Subsystem == "" {
		opts.Subsystem = subsystem
	}
	opts.ConstLabels = setMirPinnedTags(opts.ConstLabels)

	return prometheus.NewCounter(opts)
}

// NewGauge creates a new Prometheus Gauge metric with the given options.
func NewGauge(opts prometheus.GaugeOpts) prometheus.Gauge {
	if opts.Namespace == "" {
		opts.Namespace = namespace
	}
	if opts.Subsystem == "" {
		opts.Subsystem = subsystem
	}
	opts.ConstLabels = setMirPinnedTags(opts.ConstLabels)

	return prometheus.NewGauge(opts)
}

// NewHistogram creates a new Prometheus Histogram metric with the given options.
func NewHistogram(opts prometheus.HistogramOpts) prometheus.Histogram {
	if opts.Namespace == "" {
		opts.Namespace = namespace
	}
	if opts.Subsystem == "" {
		opts.Subsystem = subsystem
	}
	opts.ConstLabels = setMirPinnedTags(opts.ConstLabels)

	return prometheus.NewHistogram(opts)
}

// NewSummary creates a new Prometheus Summary metric with the given options.
func NewSummary(opts prometheus.SummaryOpts) prometheus.Summary {
	if opts.Namespace == "" {
		opts.Namespace = namespace
	}
	if opts.Subsystem == "" {
		opts.Subsystem = subsystem
	}
	opts.ConstLabels = setMirPinnedTags(opts.ConstLabels)

	return prometheus.NewSummary(opts)
}

func Register(c prometheus.Collector) {
	registry.Register(c)
}

// RegisterRoutes registers the /metrics route for exposing the Prometheus metrics.
func RegisterRoutes(r *http.ServeMux) {
	r.Handle("/metrics", promhttp.Handler())
}

// setMirNomenclature sets the Mir-specific nomenclature for the given Prometheus options.
func setMirPinnedTags(tags prometheus.Labels) prometheus.Labels {
	if tags == nil {
		tags = prometheus.Labels{}
	}
	for k, v := range extraTags {
		tags[k] = v
	}
	return tags
}
