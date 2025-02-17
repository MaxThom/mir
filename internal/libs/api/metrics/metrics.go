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
	extraTags = map[string]string{}
)

func init() {
	// Register default collectors
	registry.Register(collectors.NewBuildInfoCollector())
	registry.Register(collectors.NewGoCollector())
	registry.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	prometheus.DefaultGatherer = registry
	prometheus.DefaultRegisterer = registry
}

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
	// set module vars
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

}

func Registry() *prometheus.Registry {
	return registry
}

// NewCounter creates a new Prometheus Counter metric with the given options.
func NewCounter(opts prometheus.CounterOpts) prometheus.Counter {
	if opts.Namespace == "" {
		opts.Namespace = namespace
	}
	opts.ConstLabels = setPinnedTags(opts.ConstLabels)

	c := prometheus.NewCounter(opts)
	c.Add(0)
	registry.Register(c)
	return c
}

// NewCounterVec creates a new Prometheus CounterVec metric with the given options.
// Vector counter are not initialized with a value of 0.
func NewCounterVec(opts prometheus.CounterOpts, labels []string) *prometheus.CounterVec {
	if opts.Namespace == "" {
		opts.Namespace = namespace
	}
	opts.ConstLabels = setPinnedTags(opts.ConstLabels)

	c := prometheus.NewCounterVec(opts, labels)
	registry.Register(c)
	return c
}

// NewGauge creates a new Prometheus Gauge metric with the given options.
func NewGauge(opts prometheus.GaugeOpts) prometheus.Gauge {
	if opts.Namespace == "" {
		opts.Namespace = namespace
	}
	opts.ConstLabels = setPinnedTags(opts.ConstLabels)

	c := prometheus.NewGauge(opts)
	c.Add(0)
	registry.Register(c)
	return c
}

// NewGaugeVec creates a new Prometheus Gauge metric with the given options.
// Gauge counter are not intialized with a value of 0
func NewGaugeVec(opts prometheus.GaugeOpts, labels []string) *prometheus.GaugeVec {
	if opts.Namespace == "" {
		opts.Namespace = namespace
	}
	opts.ConstLabels = setPinnedTags(opts.ConstLabels)

	c := prometheus.NewGaugeVec(opts, labels)
	registry.Register(c)
	return c
}

// NewHistogram creates a new Prometheus Histogram metric with the given options.
func NewHistogram(opts prometheus.HistogramOpts) prometheus.Histogram {
	if opts.Namespace == "" {
		opts.Namespace = namespace
	}
	opts.ConstLabels = setPinnedTags(opts.ConstLabels)

	c := prometheus.NewHistogram(opts)
	c.Observe(0)
	registry.Register(c)
	return c
}

// NewSummary creates a new Prometheus Summary metric with the given options.
func NewSummary(opts prometheus.SummaryOpts) prometheus.Summary {
	if opts.Namespace == "" {
		opts.Namespace = namespace
	}
	opts.ConstLabels = setPinnedTags(opts.ConstLabels)

	c := prometheus.NewSummary(opts)
	c.Observe(0)
	registry.Register(c)
	return c
}

func Register(c prometheus.Collector) {
	registry.MustRegister(c)
}

func Registers(cs ...prometheus.Collector) {
	for _, c := range cs {
		registry.MustRegister(c)
	}
}

// RegisterRoutes registers the /metrics route for exposing the Prometheus metrics.
func RegisterRoutes(r *http.ServeMux) {
	r.Handle("/metrics", promhttp.Handler())
}

// setMirNomenclature sets the Mir-specific nomenclature for the given Prometheus options.
func setPinnedTags(tags prometheus.Labels) prometheus.Labels {
	if tags == nil {
		tags = prometheus.Labels{}
	}
	for k, v := range extraTags {
		tags[k] = v
	}
	return tags
}
