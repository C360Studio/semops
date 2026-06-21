package componentmetrics

import (
	"errors"
	"fmt"
	"time"

	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/metric"
	"github.com/prometheus/client_golang/prometheus"
)

// Source binds a running SemStreams component to the product feed and role
// labels operators use when reading SemOps runtime metrics.
type Source struct {
	Feed      string
	Role      string
	Component component.Discoverable
}

// SourceProvider supplies the currently running SemOps components.
type SourceProvider interface {
	ComponentMetricSources() []Source
}

// SourceProviderFunc adapts a function into a SourceProvider.
type SourceProviderFunc func() []Source

func (f SourceProviderFunc) ComponentMetricSources() []Source {
	if f == nil {
		return nil
	}
	return f()
}

// Register adds the SemOps component collector to a SemStreams metrics registry.
func Register(registry *metric.MetricsRegistry, provider SourceProvider) error {
	if registry == nil {
		return fmt.Errorf("component metrics registry is required")
	}
	if provider == nil {
		return fmt.Errorf("component metrics source provider is required")
	}
	if err := registry.PrometheusRegistry().Register(NewCollector(provider)); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			return nil
		}
		return fmt.Errorf("register component metrics collector: %w", err)
	}
	return nil
}

// Collector exports component Health and DataFlow values as Prometheus samples.
type Collector struct {
	provider SourceProvider

	info              *prometheus.Desc
	healthStatus      *prometheus.Desc
	healthLastCheck   *prometheus.Desc
	uptime            *prometheus.Desc
	errorCount        *prometheus.Desc
	messagesPerSecond *prometheus.Desc
	bytesPerSecond    *prometheus.Desc
	errorRate         *prometheus.Desc
	lastActivity      *prometheus.Desc
}

// NewCollector creates a Prometheus collector over SemStreams Discoverable components.
func NewCollector(provider SourceProvider) *Collector {
	baseLabels := []string{"component", "feed", "role", "type"}
	return &Collector{
		provider: provider,
		info: prometheus.NewDesc(
			"semops_component_info",
			"SemOps component metadata.",
			[]string{"component", "feed", "role", "type", "version"},
			nil,
		),
		healthStatus: prometheus.NewDesc(
			"semops_component_health_status",
			"SemOps component health status (1=healthy, 0=unhealthy).",
			[]string{"component", "feed", "role", "type", "status"},
			nil,
		),
		healthLastCheck: prometheus.NewDesc(
			"semops_component_health_last_check_timestamp_seconds",
			"Unix timestamp for the component's last health check.",
			baseLabels,
			nil,
		),
		uptime: prometheus.NewDesc(
			"semops_component_uptime_seconds",
			"Component uptime in seconds.",
			baseLabels,
			nil,
		),
		errorCount: prometheus.NewDesc(
			"semops_component_error_count_total",
			"Total component errors reported by SemStreams Health.",
			baseLabels,
			nil,
		),
		messagesPerSecond: prometheus.NewDesc(
			"semops_component_flow_messages_per_second",
			"Component message throughput from SemStreams DataFlow.",
			baseLabels,
			nil,
		),
		bytesPerSecond: prometheus.NewDesc(
			"semops_component_flow_bytes_per_second",
			"Component byte throughput from SemStreams DataFlow.",
			baseLabels,
			nil,
		),
		errorRate: prometheus.NewDesc(
			"semops_component_flow_error_rate",
			"Component error rate from SemStreams DataFlow.",
			baseLabels,
			nil,
		),
		lastActivity: prometheus.NewDesc(
			"semops_component_flow_last_activity_timestamp_seconds",
			"Unix timestamp for the component's last data-flow activity.",
			baseLabels,
			nil,
		),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.info
	ch <- c.healthStatus
	ch <- c.healthLastCheck
	ch <- c.uptime
	ch <- c.errorCount
	ch <- c.messagesPerSecond
	ch <- c.bytesPerSecond
	ch <- c.errorRate
	ch <- c.lastActivity
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	if c == nil || c.provider == nil {
		return
	}
	for _, source := range c.provider.ComponentMetricSources() {
		if source.Component == nil {
			continue
		}
		meta := source.Component.Meta()
		health := source.Component.Health()
		flow := source.Component.DataFlow()

		componentName := labelValue(meta.Name, "unknown")
		feed := labelValue(source.Feed, "unknown")
		role := labelValue(source.Role, "unknown")
		componentType := labelValue(meta.Type, "unknown")
		version := labelValue(meta.Version, "unknown")
		status := labelValue(health.Status, "unknown")
		base := []string{componentName, feed, role, componentType}

		ch <- prometheus.MustNewConstMetric(c.info, prometheus.GaugeValue, 1, componentName, feed, role, componentType, version)
		ch <- prometheus.MustNewConstMetric(c.healthStatus, prometheus.GaugeValue, boolFloat(health.Healthy), componentName, feed, role, componentType, status)
		ch <- prometheus.MustNewConstMetric(c.healthLastCheck, prometheus.GaugeValue, unixSeconds(health.LastCheck), base...)
		ch <- prometheus.MustNewConstMetric(c.uptime, prometheus.GaugeValue, durationSeconds(health.Uptime), base...)
		ch <- prometheus.MustNewConstMetric(c.errorCount, prometheus.CounterValue, float64(health.ErrorCount), base...)
		ch <- prometheus.MustNewConstMetric(c.messagesPerSecond, prometheus.GaugeValue, flow.MessagesPerSecond, base...)
		ch <- prometheus.MustNewConstMetric(c.bytesPerSecond, prometheus.GaugeValue, flow.BytesPerSecond, base...)
		ch <- prometheus.MustNewConstMetric(c.errorRate, prometheus.GaugeValue, flow.ErrorRate, base...)
		ch <- prometheus.MustNewConstMetric(c.lastActivity, prometheus.GaugeValue, unixSeconds(flow.LastActivity), base...)
	}
}

func labelValue(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func boolFloat(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func unixSeconds(value time.Time) float64 {
	if value.IsZero() {
		return 0
	}
	return float64(value.UnixNano()) / float64(time.Second)
}

func durationSeconds(value time.Duration) float64 {
	if value < 0 {
		return 0
	}
	return value.Seconds()
}
