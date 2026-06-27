package semconnect

import (
	"context"
	"time"

	copapi "github.com/c360studio/semops/internal/api/cop"
	readmodel "github.com/c360studio/semops/internal/egress/csapi"
)

const (
	DefaultReadSideFixtureID = "semops.csapi.read-side.hadr.v1"

	FixtureStatusPlanned = "planned"
	FixtureStatusPassed  = "passed"
	FixtureStatusFailed  = "failed"
)

var defaultFixtureObservedAt = time.Date(2026, 6, 27, 14, 15, 0, 0, time.UTC)

type FixtureOption func(*fixtureConfig)

type fixtureConfig struct {
	observedAt     time.Time
	snapshot       *copapi.Snapshot
	planOptions    []Option
	executeOptions []ExecuteOption
}

type ReadSideFixtureResult struct {
	FixtureID             string                      `json:"fixture_id"`
	Status                string                      `json:"status"`
	StartedAt             time.Time                   `json:"started_at,omitempty"`
	FinishedAt            time.Time                   `json:"finished_at,omitempty"`
	BaseURL               string                      `json:"base_url,omitempty"`
	ClaimScope            string                      `json:"claim_scope"`
	CatalogCounts         ResourceCounts              `json:"catalog_counts"`
	PlanCounts            ResourceCounts              `json:"plan_counts"`
	Catalog               readmodel.Catalog           `json:"catalog"`
	Plan                  ReadSidePlan                `json:"plan"`
	Execution             ExecuteResult               `json:"execution,omitempty"`
	Deferred              []readmodel.DeferredSurface `json:"deferred_surfaces,omitempty"`
	Error                 string                      `json:"error,omitempty"`
	ConformanceAcceptance string                      `json:"conformance_acceptance"`
}

type ResourceCounts struct {
	Systems      int `json:"systems,omitempty"`
	Deployments  int `json:"deployments,omitempty"`
	Datastreams  int `json:"datastreams,omitempty"`
	Observations int `json:"observations,omitempty"`
	SystemEvents int `json:"system_events,omitempty"`
	Requests     int `json:"requests,omitempty"`
}

func WithFixtureObservedAt(observedAt time.Time) FixtureOption {
	return func(cfg *fixtureConfig) {
		cfg.observedAt = observedAt
	}
}

func WithFixtureSnapshot(snapshot copapi.Snapshot) FixtureOption {
	return func(cfg *fixtureConfig) {
		cfg.snapshot = &snapshot
	}
}

func WithFixturePlanOptions(options ...Option) FixtureOption {
	return func(cfg *fixtureConfig) {
		cfg.planOptions = append(cfg.planOptions, options...)
	}
}

func WithFixtureExecuteOptions(options ...ExecuteOption) FixtureOption {
	return func(cfg *fixtureConfig) {
		cfg.executeOptions = append(cfg.executeOptions, options...)
	}
}

func PlanReadSideFixture(options ...FixtureOption) (ReadSideFixtureResult, error) {
	cfg := newFixtureConfig(options...)
	snapshot := cfg.readSideSnapshot()
	catalog := readmodel.ProjectSnapshot(snapshot)
	plan, err := BuildReadSidePlan(catalog, cfg.planOptions...)
	if err != nil {
		return ReadSideFixtureResult{
			FixtureID:     DefaultReadSideFixtureID,
			Status:        FixtureStatusFailed,
			ClaimScope:    catalog.ClaimScope,
			CatalogCounts: countCatalog(catalog),
			Catalog:       catalog,
			Error:         err.Error(),
		}, err
	}
	return fixtureResult(FixtureStatusPlanned, "", catalog, plan), nil
}

func RunReadSideFixture(ctx context.Context, baseURL string, options ...FixtureOption) (ReadSideFixtureResult, error) {
	cfg := newFixtureConfig(options...)
	snapshot := cfg.readSideSnapshot()
	catalog := readmodel.ProjectSnapshot(snapshot)
	plan, err := BuildReadSidePlan(catalog, cfg.planOptions...)
	if err != nil {
		result := fixtureResult(FixtureStatusFailed, baseURL, catalog, ReadSidePlan{})
		result.Error = err.Error()
		return result, err
	}

	result := fixtureResult(FixtureStatusPlanned, baseURL, catalog, plan)
	execution, err := ExecuteReadSidePlan(ctx, baseURL, plan, cfg.executeOptions...)
	result.Execution = execution
	result.StartedAt = execution.StartedAt
	result.FinishedAt = execution.FinishedAt
	if err != nil {
		result.Status = FixtureStatusFailed
		result.Error = err.Error()
		return result, err
	}
	result.Status = FixtureStatusPassed
	return result, nil
}

func newFixtureConfig(options ...FixtureOption) fixtureConfig {
	cfg := fixtureConfig{observedAt: defaultFixtureObservedAt}
	for _, opt := range options {
		opt(&cfg)
	}
	if cfg.observedAt.IsZero() {
		cfg.observedAt = defaultFixtureObservedAt
	}
	cfg.observedAt = cfg.observedAt.UTC()
	return cfg
}

func (cfg fixtureConfig) readSideSnapshot() copapi.Snapshot {
	if cfg.snapshot != nil {
		return *cfg.snapshot
	}
	return defaultReadSideFixtureSnapshot(cfg.observedAt)
}

func fixtureResult(status, baseURL string, catalog readmodel.Catalog, plan ReadSidePlan) ReadSideFixtureResult {
	return ReadSideFixtureResult{
		FixtureID:             DefaultReadSideFixtureID,
		Status:                status,
		BaseURL:               baseURL,
		ClaimScope:            catalog.ClaimScope,
		CatalogCounts:         countCatalog(catalog),
		PlanCounts:            countPlan(plan),
		Catalog:               catalog,
		Plan:                  plan,
		Deferred:              append([]readmodel.DeferredSurface(nil), catalog.DeferredSurfaces...),
		ConformanceAcceptance: "SemConnect ETS ./conformance/run.sh remains the conformance acceptance gate; this fixture is read-side bridge evidence only.",
	}
}

func countCatalog(catalog readmodel.Catalog) ResourceCounts {
	return ResourceCounts{
		Systems:      len(catalog.Systems),
		Deployments:  len(catalog.Deployments),
		Datastreams:  len(catalog.Datastreams),
		Observations: len(catalog.Observations),
		SystemEvents: len(catalog.SystemEvents),
	}
}

func countPlan(plan ReadSidePlan) ResourceCounts {
	counts := ResourceCounts{Requests: len(plan.Requests)}
	for _, request := range plan.Requests {
		switch request.Resource {
		case ResourceSystem:
			counts.Systems++
		case ResourceDeployment:
			counts.Deployments++
		case ResourceDatastream:
			counts.Datastreams++
		case ResourceObservation:
			counts.Observations++
		case ResourceSystemEvent:
			counts.SystemEvents++
		}
	}
	return counts
}

func defaultReadSideFixtureSnapshot(observedAt time.Time) copapi.Snapshot {
	assetID := "c360.edge.cop.mavlink.asset.system-42"
	trackID := "c360.edge.cop.mavlink.track.system-42"
	generatedAt := observedAt.Add(2 * time.Second)
	return copapi.Snapshot{
		GeneratedAt: generatedAt,
		Scenario:    "csapi-read-side-hadr-fixture",
		Summary: copapi.Summary{
			ActiveTracks: 1,
			ActiveAlerts: 1,
		},
		Assets: []copapi.Asset{{
			ID:         assetID,
			Label:      "MAVLink system 42",
			Kind:       "mavlink-system",
			Source:     "mavlink",
			Position:   &copapi.GeoPoint{Lat: 38.9001, Lon: -77.0002},
			Confidence: 1,
			UpdatedAt:  observedAt,
			Provenance: copapi.Provenance{
				Owner:     "semops.feed.mavlink",
				SourceRef: "mavlink://fixture/read-side/system-42",
				Observed:  observedAt,
			},
		}},
		Tracks: []copapi.Track{{
			ID:         trackID,
			Label:      "UAS 42",
			Source:     "mavlink",
			Status:     "active.armed",
			Position:   copapi.GeoPoint{Lat: 38.9001, Lon: -77.0002},
			Velocity:   "NED_CMPS(321 -12 7)",
			Confidence: 0.99,
			UpdatedAt:  observedAt,
			Provenance: copapi.Provenance{
				Owner:     "semops.feed.mavlink",
				SourceRef: "mavlink://fixture/read-side/track-system-42",
				Observed:  observedAt,
			},
		}},
		Alerts: []copapi.Alert{{
			ID:        "alert.mavlink.track-freshness",
			Label:     "Track freshness nominal",
			Severity:  "info",
			Status:    "active",
			EntityID:  trackID,
			Reason:    "MAVLink position observed within freshness window",
			UpdatedAt: observedAt,
		}},
	}
}
