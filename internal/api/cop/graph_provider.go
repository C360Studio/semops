package cop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	copmodel "github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
)

const (
	SubjectGraphQueryEntity  = "graph.query.entity"
	DefaultGraphQueryTimeout = 2 * time.Second
)

type GraphRequester interface {
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
}

type classifiedGraphRequester interface {
	RequestClassified(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
}

type GraphProvider struct {
	requester      GraphRequester
	querySubject   string
	queryTimeout   time.Duration
	now            func() time.Time
	fallback       SnapshotProvider
	mavlinkSystems []MAVLinkSystemRef
}

type MAVLinkSystemRef struct {
	Org      string
	Platform string
	SystemID int
}

type GraphProviderOption func(*GraphProvider)

func NewGraphProvider(requester GraphRequester, opts ...GraphProviderOption) (*GraphProvider, error) {
	if requester == nil {
		return nil, fmt.Errorf("graph snapshot provider requires a requester")
	}
	provider := &GraphProvider{
		requester:    requester,
		querySubject: SubjectGraphQueryEntity,
		queryTimeout: DefaultGraphQueryTimeout,
		now:          time.Now,
		mavlinkSystems: []MAVLinkSystemRef{{
			Org:      "c360",
			Platform: "edge",
			SystemID: 42,
		}},
	}
	for _, opt := range opts {
		opt(provider)
	}
	provider.mavlinkSystems = normalizeMAVLinkSystems(provider.mavlinkSystems)
	if provider.now == nil {
		provider.now = time.Now
	}
	if provider.queryTimeout <= 0 {
		provider.queryTimeout = DefaultGraphQueryTimeout
	}
	if provider.querySubject == "" {
		provider.querySubject = SubjectGraphQueryEntity
	}
	return provider, nil
}

func WithGraphFallback(provider SnapshotProvider) GraphProviderOption {
	return func(graphProvider *GraphProvider) {
		graphProvider.fallback = provider
	}
}

func WithGraphQueryTimeout(timeout time.Duration) GraphProviderOption {
	return func(provider *GraphProvider) {
		if timeout > 0 {
			provider.queryTimeout = timeout
		}
	}
}

func WithGraphNow(now func() time.Time) GraphProviderOption {
	return func(provider *GraphProvider) {
		if now != nil {
			provider.now = now
		}
	}
}

func WithMAVLinkSystems(org, platform string, systemIDs []int) GraphProviderOption {
	return func(provider *GraphProvider) {
		provider.mavlinkSystems = make([]MAVLinkSystemRef, 0, len(systemIDs))
		for _, systemID := range systemIDs {
			provider.mavlinkSystems = append(provider.mavlinkSystems, MAVLinkSystemRef{
				Org:      org,
				Platform: platform,
				SystemID: systemID,
			})
		}
	}
}

func (p *GraphProvider) Snapshot(ctx context.Context) (Snapshot, error) {
	assetsByID := make(map[string]graph.EntityState)
	tracksByID := make(map[string]graph.EntityState)
	var firstErr error

	for _, system := range p.mavlinkSystems {
		if asset, ok, err := p.queryEntity(ctx, system.assetID()); err != nil {
			firstErr = errors.Join(firstErr, err)
		} else if ok {
			assetsByID[asset.ID] = asset
		}
		if track, ok, err := p.queryEntity(ctx, system.trackID()); err != nil {
			firstErr = errors.Join(firstErr, err)
		} else if ok {
			tracksByID[track.ID] = track
		}
	}

	if len(assetsByID) == 0 && len(tracksByID) == 0 {
		if p.fallback != nil {
			return p.fallback.Snapshot(ctx)
		}
		if firstErr != nil {
			return Snapshot{}, firstErr
		}
	}

	return p.snapshotFromGraph(assetsByID, tracksByID), nil
}

func (p *GraphProvider) queryEntity(ctx context.Context, entityID string) (graph.EntityState, bool, error) {
	body, err := json.Marshal(map[string]string{"id": entityID})
	if err != nil {
		return graph.EntityState{}, false, err
	}

	response, err := p.requestEntity(ctx, body)
	if err != nil {
		if isNotFoundText(err.Error()) {
			return graph.EntityState{}, false, nil
		}
		return graph.EntityState{}, false, fmt.Errorf("query graph entity %s: %w", entityID, err)
	}
	if isLegacyErrorResponse(response) {
		if isNotFoundText(string(response)) {
			return graph.EntityState{}, false, nil
		}
		return graph.EntityState{}, false, fmt.Errorf("query graph entity %s: %s", entityID, string(response))
	}

	var entity graph.EntityState
	if err := json.Unmarshal(response, &entity); err != nil {
		return graph.EntityState{}, false, fmt.Errorf("decode graph entity %s: %w", entityID, err)
	}
	if entity.ID == "" {
		return graph.EntityState{}, false, nil
	}
	return entity, true, nil
}

func (p *GraphProvider) requestEntity(ctx context.Context, body []byte) ([]byte, error) {
	if classified, ok := p.requester.(classifiedGraphRequester); ok {
		return classified.RequestClassified(ctx, p.querySubject, body, p.queryTimeout)
	}
	return p.requester.Request(ctx, p.querySubject, body, p.queryTimeout)
}

func (p *GraphProvider) snapshotFromGraph(
	assetsByID map[string]graph.EntityState,
	tracksByID map[string]graph.EntityState,
) Snapshot {
	now := p.now().UTC()
	trackSourcePositions := make(map[string]GeoPoint)
	tracks := make([]Track, 0, len(tracksByID))
	for _, entity := range sortedEntities(tracksByID) {
		track, sourceID, ok := trackFromEntity(entity)
		if !ok {
			continue
		}
		tracks = append(tracks, track)
		if sourceID != "" {
			trackSourcePositions[sourceID] = track.Position
		}
	}

	assets := make([]Asset, 0, len(assetsByID))
	for _, entity := range sortedEntities(assetsByID) {
		position, hasPosition := trackSourcePositions[entity.ID]
		var point *GeoPoint
		if hasPosition {
			point = &position
		}
		assets = append(assets, assetFromEntity(entity, point))
	}

	feeds := firstPhaseFeedHealth(now, tracks)
	return Snapshot{
		GeneratedAt: now,
		Scenario:    "phase-1-live-graph",
		Summary: Summary{
			ActiveTracks: len(tracks),
			ActiveAlerts: 0,
			StaleFeeds:   countFeeds(feeds, "stale"),
		},
		Feeds:   feeds,
		Assets:  assets,
		Tracks:  tracks,
		Hazards: []Hazard{},
		Alerts:  []Alert{},
	}
}

func firstPhaseFeedHealth(now time.Time, tracks []Track) []FeedHealth {
	mavlink := FeedHealth{
		ID:          "feed.mavlink",
		Name:        "MAVLink",
		Kind:        "telemetry",
		Status:      "stale",
		LastEventAt: now,
		Message:     "Waiting for graph-backed MAVLink state",
	}
	if len(tracks) > 0 {
		mavlink.Status = "live"
		mavlink.LastEventAt = latestTrackUpdate(tracks)
		mavlink.Message = "Graph-backed source asset and track state"
	}
	return []FeedHealth{
		mavlink,
		{
			ID:          "feed.tak",
			Name:        "TAK/CoT",
			Kind:        "operators",
			Status:      "planned",
			LastEventAt: now.Add(-18 * time.Minute),
			Message:     "Seed replay gate pending",
		},
		{
			ID:          "feed.cap",
			Name:        "CAP",
			Kind:        "advisory",
			Status:      "planned",
			LastEventAt: now.Add(-33 * time.Minute),
			Message:     "Schema/sample gate pending",
		},
	}
}

func latestTrackUpdate(tracks []Track) time.Time {
	var latest time.Time
	for _, track := range tracks {
		if track.UpdatedAt.After(latest) {
			latest = track.UpdatedAt
		}
	}
	return latest
}

func assetFromEntity(entity graph.EntityState, position *GeoPoint) Asset {
	updatedAt := observedAt(entity, copmodel.ProvenanceObservedAt)
	return Asset{
		ID:         entity.ID,
		Label:      stringProperty(entity, copmodel.AssetName, instanceLabel(entity.ID)),
		Kind:       stringProperty(entity, copmodel.AssetKind, "asset"),
		Source:     stringProperty(entity, copmodel.AssetSource, "graph"),
		Position:   position,
		Confidence: confidence(entity),
		UpdatedAt:  updatedAt,
		Provenance: Provenance{
			Owner:     copmodel.OwnerAsset,
			SourceRef: stringProperty(entity, copmodel.ProvenanceSourceRef, ""),
			Observed:  updatedAt,
		},
	}
}

func trackFromEntity(entity graph.EntityState) (Track, string, bool) {
	positionValue, ok := propertyValue(entity, copmodel.TrackPosition)
	if !ok {
		return Track{}, "", false
	}
	position, ok := geoPointFromWKT(positionValue)
	if !ok {
		return Track{}, "", false
	}

	sourceID := stringProperty(entity, copmodel.TrackSource, "")
	updatedAt := observedAt(entity, copmodel.TrackObservedAt, copmodel.ProvenanceObservedAt)
	return Track{
		ID:         entity.ID,
		Label:      trackLabel(entity),
		Source:     stringProperty(entity, copmodel.ProvenanceSource, "mavlink"),
		Status:     stringProperty(entity, copmodel.TrackStatus, "unknown"),
		Position:   position,
		Velocity:   stringProperty(entity, copmodel.TrackVelocity, ""),
		Confidence: confidence(entity),
		UpdatedAt:  updatedAt,
		Provenance: Provenance{
			Owner:     copmodel.OwnerMAVLink,
			SourceRef: stringProperty(entity, copmodel.ProvenanceSourceRef, ""),
			Observed:  updatedAt,
		},
	}, sourceID, true
}

func sortedEntities(entities map[string]graph.EntityState) []graph.EntityState {
	ids := make([]string, 0, len(entities))
	for id := range entities {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]graph.EntityState, 0, len(ids))
	for _, id := range ids {
		out = append(out, entities[id])
	}
	return out
}

func propertyValue(entity graph.EntityState, predicate string) (any, bool) {
	return (&entity).GetPropertyValue(predicate)
}

func stringProperty(entity graph.EntityState, predicate string, fallback string) string {
	value, ok := propertyValue(entity, predicate)
	if !ok {
		return fallback
	}
	text, ok := stringFromAny(value)
	if !ok || text == "" {
		return fallback
	}
	return text
}

func confidence(entity graph.EntityState) float64 {
	if value, ok := propertyValue(entity, copmodel.ProvenanceConfidence); ok {
		if parsed, ok := floatFromAny(value); ok {
			return parsed
		}
	}
	for _, triple := range entity.Triples {
		if triple.Confidence > 0 {
			return triple.Confidence
		}
	}
	return 1
}

func observedAt(entity graph.EntityState, predicates ...string) time.Time {
	for _, predicate := range predicates {
		if value, ok := propertyValue(entity, predicate); ok {
			if parsed, ok := timeFromAny(value); ok {
				return parsed.UTC()
			}
		}
	}
	latest := entity.UpdatedAt
	for _, triple := range entity.Triples {
		if triple.Timestamp.After(latest) {
			latest = triple.Timestamp
		}
	}
	if latest.IsZero() {
		return time.Now().UTC()
	}
	return latest.UTC()
}

func geoPointFromWKT(value any) (GeoPoint, bool) {
	text, ok := stringFromAny(value)
	if !ok {
		return GeoPoint{}, false
	}
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "POINT(") || !strings.HasSuffix(text, ")") {
		return GeoPoint{}, false
	}
	parts := strings.Fields(strings.TrimSuffix(strings.TrimPrefix(text, "POINT("), ")"))
	if len(parts) != 2 {
		return GeoPoint{}, false
	}
	lon, lonErr := strconv.ParseFloat(parts[0], 64)
	lat, latErr := strconv.ParseFloat(parts[1], 64)
	if lonErr != nil || latErr != nil {
		return GeoPoint{}, false
	}
	return GeoPoint{Lat: lat, Lon: lon}, true
}

func stringFromAny(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case fmt.Stringer:
		return typed.String(), true
	default:
		return "", false
	}
}

func floatFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func timeFromAny(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed, !typed.IsZero()
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, typed)
		return parsed, err == nil
	default:
		return time.Time{}, false
	}
}

func trackLabel(entity graph.EntityState) string {
	nativeID := stringProperty(entity, copmodel.TrackNativeID, "")
	if systemID := systemIDFromNativeID(nativeID); systemID != "" {
		return "UAS " + systemID
	}
	if systemID := systemIDFromEntityID(entity.ID); systemID != "" {
		return "UAS " + systemID
	}
	return instanceLabel(entity.ID)
}

func systemIDFromNativeID(nativeID string) string {
	parts := strings.Split(nativeID, ".")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "system" && parts[i+1] != "" {
			return parts[i+1]
		}
	}
	return ""
}

func systemIDFromEntityID(entityID string) string {
	instance := instanceLabel(entityID)
	return strings.TrimPrefix(instance, "system-")
}

func instanceLabel(entityID string) string {
	parts := strings.Split(entityID, ".")
	if len(parts) == 0 {
		return entityID
	}
	return parts[len(parts)-1]
}

func isLegacyErrorResponse(response []byte) bool {
	return bytes.HasPrefix(response, []byte("error: "))
}

func isNotFoundText(text string) bool {
	return strings.Contains(strings.ToLower(text), "not found")
}

func normalizeMAVLinkSystems(systems []MAVLinkSystemRef) []MAVLinkSystemRef {
	if len(systems) == 0 {
		systems = []MAVLinkSystemRef{{Org: "c360", Platform: "edge", SystemID: 42}}
	}
	out := make([]MAVLinkSystemRef, 0, len(systems))
	for _, system := range systems {
		if system.Org == "" {
			system.Org = "c360"
		}
		if system.Platform == "" {
			system.Platform = "edge"
		}
		if system.SystemID < 0 || system.SystemID > 255 {
			continue
		}
		out = append(out, system)
	}
	if len(out) == 0 {
		out = append(out, MAVLinkSystemRef{Org: "c360", Platform: "edge", SystemID: 42})
	}
	return out
}

func (s MAVLinkSystemRef) assetID() string {
	return mavlinkEntityID(s.Org, s.Platform, copmodel.EntityAsset, s.SystemID)
}

func (s MAVLinkSystemRef) trackID() string {
	return mavlinkEntityID(s.Org, s.Platform, copmodel.EntityTrack, s.SystemID)
}

func mavlinkEntityID(org, platform, entityType string, systemID int) string {
	return message.EntityID{
		Org:      org,
		Platform: platform,
		Domain:   "cop",
		System:   "mavlink",
		Type:     entityType,
		Instance: fmt.Sprintf("system-%d", systemID),
	}.Key()
}
