package cop

import (
	"math"
	"sort"
	"time"

	"github.com/c360studio/semops/internal/componentmetrics"
)

type RuntimeProvider interface {
	ComponentMetricSources() []componentmetrics.Source
}

type RuntimeSnapshot struct {
	GeneratedAt time.Time          `json:"generated_at"`
	Feeds       []RuntimeFeed      `json:"feeds"`
	Components  []RuntimeComponent `json:"components"`
}

type RuntimeFeed struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Status              string     `json:"status"`
	Message             string     `json:"message"`
	HealthyComponents   int        `json:"healthy_components"`
	TotalComponents     int        `json:"total_components"`
	MessagesPerSecond   float64    `json:"messages_per_second"`
	LastActivity        *time.Time `json:"last_activity,omitempty"`
	LastActivityAgeSecs *int       `json:"last_activity_age_seconds,omitempty"`
}

type RuntimeComponent struct {
	Name              string     `json:"name"`
	Feed              string     `json:"feed"`
	Role              string     `json:"role"`
	Type              string     `json:"type"`
	Status            string     `json:"status"`
	Healthy           bool       `json:"healthy"`
	MessagesPerSecond float64    `json:"messages_per_second"`
	BytesPerSecond    float64    `json:"bytes_per_second"`
	ErrorRate         float64    `json:"error_rate"`
	ErrorCount        int        `json:"error_count"`
	LastActivity      *time.Time `json:"last_activity,omitempty"`
	LastCheck         *time.Time `json:"last_check,omitempty"`
	UptimeSeconds     float64    `json:"uptime_seconds"`
}

func BuildRuntimeSnapshot(now time.Time, provider RuntimeProvider) RuntimeSnapshot {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	if provider == nil {
		return RuntimeSnapshot{GeneratedAt: now}
	}

	sources := provider.ComponentMetricSources()
	components := make([]RuntimeComponent, 0, len(sources))
	feedSummaries := map[string]*runtimeFeedSummary{}
	for _, source := range sources {
		if source.Component == nil {
			continue
		}
		meta := source.Component.Meta()
		health := source.Component.Health()
		flow := source.Component.DataFlow()
		feedID, feedName := runtimeFeedLabels(source.Feed)
		component := RuntimeComponent{
			Name:              firstNonEmpty(meta.Name, "unknown"),
			Feed:              feedID,
			Role:              firstNonEmpty(source.Role, "unknown"),
			Type:              firstNonEmpty(meta.Type, "unknown"),
			Status:            firstNonEmpty(health.Status, "unknown"),
			Healthy:           health.Healthy,
			MessagesPerSecond: roundMetric(flow.MessagesPerSecond),
			BytesPerSecond:    roundMetric(flow.BytesPerSecond),
			ErrorRate:         roundMetric(flow.ErrorRate),
			ErrorCount:        health.ErrorCount,
			LastActivity:      timePtr(flow.LastActivity),
			LastCheck:         timePtr(health.LastCheck),
			UptimeSeconds:     roundMetric(health.Uptime.Seconds()),
		}
		components = append(components, component)

		summary := feedSummaries[feedID]
		if summary == nil {
			summary = &runtimeFeedSummary{id: feedID, name: feedName}
			feedSummaries[feedID] = summary
		}
		summary.total++
		if component.Healthy {
			summary.healthy++
		}
		summary.messagesPerSecond += component.MessagesPerSecond
		if component.LastActivity != nil && (summary.lastActivity == nil || component.LastActivity.After(*summary.lastActivity)) {
			activity := component.LastActivity.UTC()
			summary.lastActivity = &activity
		}
		if !component.Healthy {
			summary.degraded = true
		}
		if component.Status == "stale" {
			summary.stale = true
		}
	}

	sort.Slice(components, func(i, j int) bool {
		if components[i].Feed != components[j].Feed {
			return components[i].Feed < components[j].Feed
		}
		if components[i].Role != components[j].Role {
			return components[i].Role < components[j].Role
		}
		return components[i].Name < components[j].Name
	})

	feeds := make([]RuntimeFeed, 0, len(feedSummaries))
	for _, summary := range feedSummaries {
		feeds = append(feeds, summary.runtimeFeed(now))
	}
	sort.Slice(feeds, func(i, j int) bool {
		return runtimeFeedOrder(feeds[i].ID) < runtimeFeedOrder(feeds[j].ID)
	})

	return RuntimeSnapshot{
		GeneratedAt: now,
		Feeds:       feeds,
		Components:  components,
	}
}

type runtimeFeedSummary struct {
	id                string
	name              string
	healthy           int
	total             int
	messagesPerSecond float64
	lastActivity      *time.Time
	degraded          bool
	stale             bool
}

func (s runtimeFeedSummary) runtimeFeed(now time.Time) RuntimeFeed {
	status := "flowing"
	message := "component flow active"
	if s.stale {
		status = "stale"
		message = "source reports stale data"
	} else if s.degraded || s.healthy < s.total {
		status = "degraded"
		message = "one or more components unhealthy"
	} else if s.lastActivity == nil || s.messagesPerSecond <= 0 {
		status = "idle"
		message = "components healthy; no recent flow"
	}

	var age *int
	if s.lastActivity != nil {
		seconds := int(math.Max(0, now.Sub(*s.lastActivity).Seconds()))
		age = &seconds
	}

	return RuntimeFeed{
		ID:                  s.id,
		Name:                s.name,
		Status:              status,
		Message:             message,
		HealthyComponents:   s.healthy,
		TotalComponents:     s.total,
		MessagesPerSecond:   roundMetric(s.messagesPerSecond),
		LastActivity:        s.lastActivity,
		LastActivityAgeSecs: age,
	}
}

func runtimeFeedLabels(feed string) (string, string) {
	switch feed {
	case "mavlink":
		return "feed.mavlink", "MAVLink"
	case "tak-cot":
		return "feed.tak", "TAK/CoT"
	case "cap":
		return "feed.cap", "CAP"
	case "adsb":
		return "feed.adsb", "ADS-B"
	case "sapient":
		return "feed.sapient", "SAPIENT"
	case "klv":
		return "feed.klv", "KLV"
	default:
		if feed == "" {
			return "feed.unknown", "Unknown"
		}
		return "feed." + feed, feed
	}
}

func runtimeFeedOrder(id string) int {
	switch id {
	case "feed.mavlink":
		return 0
	case "feed.tak":
		return 1
	case "feed.adsb":
		return 2
	case "feed.cap":
		return 3
	case "feed.sapient":
		return 4
	case "feed.klv":
		return 5
	default:
		return 100
	}
}

func roundMetric(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	v := value.UTC()
	return &v
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
