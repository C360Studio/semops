package componentmetrics

import (
	"testing"
	"time"

	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/metric"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestCollectorExportsDiscoverableHealthAndFlow(t *testing.T) {
	now := time.Unix(1_700_000_000, 500_000_000).UTC()
	fake := fakeDiscoverable{
		meta: component.Metadata{
			Name:    "semops-processor-mavlink-decode",
			Type:    "processor",
			Version: "v1",
		},
		health: component.HealthStatus{
			Healthy:    true,
			LastCheck:  now,
			ErrorCount: 2,
			Uptime:     3 * time.Second,
			Status:     "started",
		},
		flow: component.FlowMetrics{
			MessagesPerSecond: 4.5,
			BytesPerSecond:    512,
			ErrorRate:         0.25,
			LastActivity:      now.Add(-time.Second),
		},
	}

	registry := prometheus.NewRegistry()
	err := registry.Register(NewCollector(SourceProviderFunc(func() []Source {
		return []Source{{Feed: "mavlink", Role: "decoder", Component: fake}}
	})))
	if err != nil {
		t.Fatalf("register collector: %v", err)
	}

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	labels := map[string]string{
		"component": "semops-processor-mavlink-decode",
		"feed":      "mavlink",
		"role":      "decoder",
		"type":      "processor",
	}
	if got := sampleValue(t, families, "semops_component_flow_messages_per_second", labels); got != 4.5 {
		t.Fatalf("messages per second = %v, want 4.5", got)
	}
	if got := sampleValue(t, families, "semops_component_flow_bytes_per_second", labels); got != 512 {
		t.Fatalf("bytes per second = %v, want 512", got)
	}
	if got := sampleValue(t, families, "semops_component_error_count_total", labels); got != 2 {
		t.Fatalf("error count = %v, want 2", got)
	}
	healthLabels := cloneLabels(labels)
	healthLabels["status"] = "started"
	if got := sampleValue(t, families, "semops_component_health_status", healthLabels); got != 1 {
		t.Fatalf("health status = %v, want 1", got)
	}
	infoLabels := cloneLabels(labels)
	infoLabels["version"] = "v1"
	if got := sampleValue(t, families, "semops_component_info", infoLabels); got != 1 {
		t.Fatalf("component info = %v, want 1", got)
	}
}

func TestRegisterUsesSemStreamsMetricsRegistry(t *testing.T) {
	registry := metric.NewMetricsRegistry()
	provider := SourceProviderFunc(func() []Source {
		return []Source{{Feed: "test", Role: "decoder", Component: fakeDiscoverable{
			meta: component.Metadata{Name: "component-a", Type: "processor", Version: "v1"},
			health: component.HealthStatus{
				Healthy:   true,
				LastCheck: time.Now().UTC(),
				Uptime:    time.Second,
				Status:    "started",
			},
			flow: component.FlowMetrics{MessagesPerSecond: 1},
		}}}
	})
	if err := Register(registry, provider); err != nil {
		t.Fatalf("register component metrics: %v", err)
	}

	families, err := registry.PrometheusRegistry().Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	labels := map[string]string{"component": "component-a", "feed": "test", "role": "decoder", "type": "processor"}
	if got := sampleValue(t, families, "semops_component_flow_messages_per_second", labels); got != 1 {
		t.Fatalf("messages per second = %v, want 1", got)
	}
}

type fakeDiscoverable struct {
	meta   component.Metadata
	health component.HealthStatus
	flow   component.FlowMetrics
}

func (f fakeDiscoverable) Meta() component.Metadata {
	return f.meta
}

func (f fakeDiscoverable) InputPorts() []component.Port {
	return nil
}

func (f fakeDiscoverable) OutputPorts() []component.Port {
	return nil
}

func (f fakeDiscoverable) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{}
}

func (f fakeDiscoverable) Health() component.HealthStatus {
	return f.health
}

func (f fakeDiscoverable) DataFlow() component.FlowMetrics {
	return f.flow
}

func sampleValue(t *testing.T, families []*dto.MetricFamily, name string, labels map[string]string) float64 {
	t.Helper()
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, sample := range family.GetMetric() {
			if metricLabelsMatch(sample.GetLabel(), labels) {
				if sample.Gauge != nil {
					return sample.Gauge.GetValue()
				}
				if sample.Counter != nil {
					return sample.Counter.GetValue()
				}
				if sample.Untyped != nil {
					return sample.Untyped.GetValue()
				}
			}
		}
	}
	t.Fatalf("missing metric %s with labels %+v", name, labels)
	return 0
}

func metricLabelsMatch(got []*dto.LabelPair, want map[string]string) bool {
	gotLabels := map[string]string{}
	for _, label := range got {
		gotLabels[label.GetName()] = label.GetValue()
	}
	for key, value := range want {
		if gotLabels[key] != value {
			return false
		}
	}
	return true
}

func cloneLabels(labels map[string]string) map[string]string {
	cloned := make(map[string]string, len(labels))
	for key, value := range labels {
		cloned[key] = value
	}
	return cloned
}
