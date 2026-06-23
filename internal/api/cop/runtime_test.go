package cop

import (
	"testing"
	"time"

	"github.com/c360studio/semops/internal/componentmetrics"
	"github.com/c360studio/semstreams/component"
)

func TestBuildRuntimeSnapshotSummarizesComponentFlow(t *testing.T) {
	now := time.Date(2026, 6, 21, 17, 30, 0, 0, time.UTC)
	lastActivity := now.Add(-8 * time.Second)
	lastCheck := now.Add(-2 * time.Second)
	provider := runtimeProviderStub{sources: []componentmetrics.Source{
		{
			Feed: "mavlink",
			Role: "input",
			Component: runtimeComponentStub{
				meta: component.Metadata{Name: "mavlink-udp", Type: "input", Version: "test"},
				health: component.HealthStatus{
					Healthy:   true,
					LastCheck: lastCheck,
					Uptime:    3 * time.Minute,
					Status:    "running",
				},
				flow: component.FlowMetrics{
					MessagesPerSecond: 3.25,
					BytesPerSecond:    512.5,
					LastActivity:      lastActivity,
				},
			},
		},
		{
			Feed: "mavlink",
			Role: "projector",
			Component: runtimeComponentStub{
				meta: component.Metadata{Name: "mavlink-projector", Type: "processor", Version: "test"},
				health: component.HealthStatus{
					Healthy:   true,
					LastCheck: lastCheck,
					Uptime:    2 * time.Minute,
					Status:    "running",
				},
				flow: component.FlowMetrics{
					MessagesPerSecond: 1.75,
					BytesPerSecond:    256,
					LastActivity:      now.Add(-12 * time.Second),
				},
			},
		},
	}}

	snapshot := BuildRuntimeSnapshot(now, provider)

	if snapshot.GeneratedAt != now {
		t.Fatalf("generated_at = %s, want %s", snapshot.GeneratedAt, now)
	}
	if len(snapshot.Feeds) != 1 {
		t.Fatalf("feed count = %d, want 1", len(snapshot.Feeds))
	}
	feed := snapshot.Feeds[0]
	if feed.ID != "feed.mavlink" || feed.Name != "MAVLink" {
		t.Fatalf("feed identity = %s/%s, want feed.mavlink/MAVLink", feed.ID, feed.Name)
	}
	if feed.Status != "flowing" {
		t.Fatalf("feed status = %q, want flowing", feed.Status)
	}
	if feed.HealthyComponents != 2 || feed.TotalComponents != 2 {
		t.Fatalf("health summary = %d/%d, want 2/2", feed.HealthyComponents, feed.TotalComponents)
	}
	if feed.MessagesPerSecond != 5 {
		t.Fatalf("messages/sec = %f, want 5", feed.MessagesPerSecond)
	}
	if feed.LastActivity == nil || !feed.LastActivity.Equal(lastActivity) {
		t.Fatalf("last activity = %v, want %s", feed.LastActivity, lastActivity)
	}
	if feed.LastActivityAgeSecs == nil || *feed.LastActivityAgeSecs != 8 {
		t.Fatalf("last activity age = %v, want 8", feed.LastActivityAgeSecs)
	}
	if len(snapshot.Components) != 2 {
		t.Fatalf("component count = %d, want 2", len(snapshot.Components))
	}
	if snapshot.Components[0].Name != "mavlink-udp" || snapshot.Components[0].UptimeSeconds != 180 {
		t.Fatalf("first component = %+v", snapshot.Components[0])
	}
}

func TestBuildRuntimeSnapshotMarksStaleBeforeGenericDegraded(t *testing.T) {
	now := time.Date(2026, 6, 21, 17, 30, 0, 0, time.UTC)
	provider := runtimeProviderStub{sources: []componentmetrics.Source{
		{
			Feed: "cap",
			Role: "http-poller",
			Component: runtimeComponentStub{
				meta: component.Metadata{Name: "cap-poller", Type: "input"},
				health: component.HealthStatus{
					Healthy: false,
					Status:  "stale",
				},
				flow: component.FlowMetrics{
					LastActivity: now.Add(-3 * time.Minute),
				},
			},
		},
		{
			Feed: "sapient",
			Role: "decoder",
			Component: runtimeComponentStub{
				meta: component.Metadata{Name: "sapient-decoder", Type: "processor"},
				health: component.HealthStatus{
					Healthy: true,
					Status:  "running",
				},
			},
		},
	}}

	snapshot := BuildRuntimeSnapshot(now, provider)

	if len(snapshot.Feeds) != 2 {
		t.Fatalf("feed count = %d, want 2", len(snapshot.Feeds))
	}
	if snapshot.Feeds[0].ID != "feed.cap" || snapshot.Feeds[0].Status != "stale" {
		t.Fatalf("first feed = %+v, want stale CAP before SAPIENT", snapshot.Feeds[0])
	}
	if snapshot.Feeds[1].ID != "feed.sapient" || snapshot.Feeds[1].Status != "idle" {
		t.Fatalf("second feed = %+v, want idle SAPIENT", snapshot.Feeds[1])
	}
}

func TestBuildRuntimeSnapshotLabelsKLVFeed(t *testing.T) {
	now := time.Date(2026, 6, 22, 19, 0, 0, 0, time.UTC)
	provider := runtimeProviderStub{sources: []componentmetrics.Source{{
		Feed: "klv",
		Role: "projector",
		Component: runtimeComponentStub{
			meta: component.Metadata{Name: "klv-projector", Type: "processor"},
			health: component.HealthStatus{
				Healthy: true,
				Status:  "running",
			},
			flow: component.FlowMetrics{
				MessagesPerSecond: 1,
				LastActivity:      now.Add(-time.Second),
			},
		},
	}}}

	snapshot := BuildRuntimeSnapshot(now, provider)

	if len(snapshot.Feeds) != 1 {
		t.Fatalf("feed count = %d, want 1", len(snapshot.Feeds))
	}
	if snapshot.Feeds[0].ID != "feed.klv" ||
		snapshot.Feeds[0].Name != "KLV" ||
		snapshot.Feeds[0].Status != "flowing" {
		t.Fatalf("KLV runtime feed = %+v", snapshot.Feeds[0])
	}
}

func TestBuildRuntimeSnapshotLabelsWeatherFeed(t *testing.T) {
	now := time.Date(2026, 6, 23, 19, 0, 0, 0, time.UTC)
	provider := runtimeProviderStub{sources: []componentmetrics.Source{{
		Feed: "weather",
		Role: "projector",
		Component: runtimeComponentStub{
			meta: component.Metadata{Name: "weather-projector", Type: "processor"},
			health: component.HealthStatus{
				Healthy: true,
				Status:  "running",
			},
			flow: component.FlowMetrics{
				MessagesPerSecond: 1,
				LastActivity:      now.Add(-time.Second),
			},
		},
	}}}

	snapshot := BuildRuntimeSnapshot(now, provider)

	if len(snapshot.Feeds) != 1 {
		t.Fatalf("feed count = %d, want 1", len(snapshot.Feeds))
	}
	if snapshot.Feeds[0].ID != "feed.weather" ||
		snapshot.Feeds[0].Name != "Weather" ||
		snapshot.Feeds[0].Status != "flowing" {
		t.Fatalf("weather runtime feed = %+v", snapshot.Feeds[0])
	}
}

type runtimeProviderStub struct {
	sources []componentmetrics.Source
}

func (p runtimeProviderStub) ComponentMetricSources() []componentmetrics.Source {
	return p.sources
}

type runtimeComponentStub struct {
	meta   component.Metadata
	health component.HealthStatus
	flow   component.FlowMetrics
}

func (c runtimeComponentStub) Meta() component.Metadata {
	return c.meta
}

func (c runtimeComponentStub) InputPorts() []component.Port {
	return nil
}

func (c runtimeComponentStub) OutputPorts() []component.Port {
	return nil
}

func (c runtimeComponentStub) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{}
}

func (c runtimeComponentStub) Health() component.HealthStatus {
	return c.health
}

func (c runtimeComponentStub) DataFlow() component.FlowMetrics {
	return c.flow
}
