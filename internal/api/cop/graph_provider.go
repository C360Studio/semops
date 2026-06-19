package cop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	capprojector "github.com/c360studio/semops/internal/projectors/cap"
	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	copmodel "github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
)

const (
	SubjectGraphQueryEntity      = "graph.query.entity"
	DefaultGraphQueryTimeout     = 2 * time.Second
	DefaultFeedFreshnessWindow   = 2 * time.Minute
	DefaultCoTUIDAndroidAlpha    = "ANDROID-ALPHA"
	DefaultCoTUIDAndroidBravo    = "ANDROID-BRAVO"
	DefaultCoTUIDMarkerNorthGate = "MARKER-NORTH-GATE"
	DefaultCoTUIDChatAlpha       = "CHAT-ALPHA-1"
	DefaultCAPAlertID            = "nws-demo-flood-warning"
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
	cotUIDs        []CoTUIDRef
	capAlerts      []CAPAlertRef
	freshness      time.Duration
}

type MAVLinkSystemRef struct {
	Org      string
	Platform string
	SystemID int
}

type CoTUIDRef struct {
	Org      string
	Platform string
	UID      string
}

type CAPAlertRef struct {
	Org        string
	Platform   string
	Identifier string
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
		freshness:    DefaultFeedFreshnessWindow,
		mavlinkSystems: []MAVLinkSystemRef{{
			Org:      "c360",
			Platform: "edge",
			SystemID: 42,
		}},
		capAlerts: []CAPAlertRef{{
			Org:        "c360",
			Platform:   "edge",
			Identifier: DefaultCAPAlertID,
		}},
	}
	for _, opt := range opts {
		opt(provider)
	}
	provider.mavlinkSystems = normalizeMAVLinkSystems(provider.mavlinkSystems)
	provider.cotUIDs = normalizeCoTUIDs(provider.cotUIDs)
	provider.capAlerts = normalizeCAPAlerts(provider.capAlerts)
	if provider.now == nil {
		provider.now = time.Now
	}
	if provider.queryTimeout <= 0 {
		provider.queryTimeout = DefaultGraphQueryTimeout
	}
	if provider.freshness <= 0 {
		provider.freshness = DefaultFeedFreshnessWindow
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

func WithCoTUIDs(org, platform string, uids []string) GraphProviderOption {
	return func(provider *GraphProvider) {
		provider.cotUIDs = make([]CoTUIDRef, 0, len(uids))
		for _, uid := range uids {
			provider.cotUIDs = append(provider.cotUIDs, CoTUIDRef{
				Org:      org,
				Platform: platform,
				UID:      uid,
			})
		}
	}
}

func WithCAPAlertIDs(org, platform string, identifiers []string) GraphProviderOption {
	return func(provider *GraphProvider) {
		provider.capAlerts = make([]CAPAlertRef, 0, len(identifiers))
		for _, identifier := range identifiers {
			provider.capAlerts = append(provider.capAlerts, CAPAlertRef{
				Org:        org,
				Platform:   platform,
				Identifier: identifier,
			})
		}
	}
}

func WithFeedFreshnessWindow(window time.Duration) GraphProviderOption {
	return func(provider *GraphProvider) {
		if window > 0 {
			provider.freshness = window
		}
	}
}

func (p *GraphProvider) Snapshot(ctx context.Context) (Snapshot, error) {
	assetsByID := make(map[string]graph.EntityState)
	tracksByID := make(map[string]graph.EntityState)
	tasksByID := make(map[string]graph.EntityState)
	advisoriesByID := make(map[string]graph.EntityState)
	hazardsByID := make(map[string]graph.EntityState)
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

	for _, uid := range p.cotUIDs {
		if asset, ok, err := p.queryEntity(ctx, uid.assetID()); err != nil {
			firstErr = errors.Join(firstErr, err)
		} else if ok {
			assetsByID[asset.ID] = asset
		}
		if track, ok, err := p.queryEntity(ctx, uid.trackID()); err != nil {
			firstErr = errors.Join(firstErr, err)
		} else if ok {
			tracksByID[track.ID] = track
		}
		if task, ok, err := p.queryEntity(ctx, uid.taskID()); err != nil {
			firstErr = errors.Join(firstErr, err)
		} else if ok {
			tasksByID[task.ID] = task
		}
		if advisory, ok, err := p.queryEntity(ctx, uid.advisoryID()); err != nil {
			firstErr = errors.Join(firstErr, err)
		} else if ok {
			advisoriesByID[advisory.ID] = advisory
		}
	}

	for _, alert := range p.capAlerts {
		if hazard, ok, err := p.queryEntity(ctx, alert.hazardID()); err != nil {
			firstErr = errors.Join(firstErr, err)
		} else if ok {
			hazardsByID[hazard.ID] = hazard
		}
	}

	if len(assetsByID) == 0 &&
		len(tracksByID) == 0 &&
		len(tasksByID) == 0 &&
		len(advisoriesByID) == 0 &&
		len(hazardsByID) == 0 {
		if p.fallback != nil {
			return p.fallback.Snapshot(ctx)
		}
		if firstErr != nil {
			return Snapshot{}, firstErr
		}
	}

	return p.snapshotFromGraph(assetsByID, tracksByID, tasksByID, advisoriesByID, hazardsByID), nil
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
	tasksByID map[string]graph.EntityState,
	advisoriesByID map[string]graph.EntityState,
	hazardsByID map[string]graph.EntityState,
) Snapshot {
	now := p.now().UTC()
	trackSourcePositions := make(map[string]GeoPoint)
	tracks := make([]Track, 0, len(tracksByID))
	for _, entity := range sortedEntities(tracksByID) {
		track, sourceID, ok := trackFromEntity(entity, now, p.freshness)
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

	tasks := make([]Task, 0, len(tasksByID))
	for _, entity := range sortedEntities(tasksByID) {
		task, ok := taskFromEntity(entity, now, p.freshness)
		if ok {
			tasks = append(tasks, task)
		}
	}

	advisories := make([]Advisory, 0, len(advisoriesByID))
	for _, entity := range sortedEntities(advisoriesByID) {
		advisory, ok := advisoryFromEntity(entity, now, p.freshness)
		if ok {
			advisories = append(advisories, advisory)
		}
	}

	hazards := make([]Hazard, 0, len(hazardsByID))
	for _, entity := range sortedEntities(hazardsByID) {
		hazard, ok := hazardFromEntity(entity)
		if ok {
			hazards = append(hazards, hazard)
		}
	}

	feeds := firstPhaseFeedHealth(now, p.freshness, tracks, tasks, advisories, hazards)
	return Snapshot{
		GeneratedAt: now,
		Scenario:    "phase-1-live-graph",
		Summary: Summary{
			ActiveTracks:     len(tracks),
			ActiveTasks:      len(tasks),
			ActiveAdvisories: len(advisories),
			ActiveAlerts:     0,
			StaleFeeds:       countFeeds(feeds, "stale"),
		},
		Feeds:      feeds,
		Assets:     assets,
		Tracks:     tracks,
		Tasks:      tasks,
		Advisories: advisories,
		Hazards:    hazards,
		Alerts:     []Alert{},
	}
}

func firstPhaseFeedHealth(
	now time.Time,
	freshness time.Duration,
	tracks []Track,
	tasks []Task,
	advisories []Advisory,
	hazards []Hazard,
) []FeedHealth {
	mavlink := feedHealthFromObservations(
		now,
		freshness,
		"feed.mavlink",
		"MAVLink",
		"telemetry",
		"Waiting for graph-backed MAVLink state",
		"Graph-backed source asset and track state",
		trackObservationTimes(filterTracksBySource(tracks, "mavlink")),
	)
	takObservations := append(
		trackObservationTimes(filterTracksBySource(tracks, "tak-cot")),
		taskObservationTimes(tasks)...,
	)
	takObservations = append(takObservations, advisoryObservationTimes(advisories)...)
	tak := feedHealthFromObservations(
		now,
		freshness,
		"feed.tak",
		"TAK/CoT",
		"operators",
		"Waiting for graph-backed TAK/CoT state",
		"Graph-backed CoT tracks, tasks, and advisories",
		takObservations,
	)
	cap := feedHealthFromObservations(
		now,
		freshness,
		"feed.cap",
		"CAP",
		"advisory",
		"Schema/sample gate pending",
		"Graph-backed civilian alert evidence",
		hazardObservationTimes(hazards),
	)
	if len(hazards) == 0 {
		cap.Status = "planned"
		cap.LastEventAt = now.Add(-33 * time.Minute)
	}
	return []FeedHealth{mavlink, tak, cap}
}

func feedHealthFromObservations(
	now time.Time,
	freshness time.Duration,
	id string,
	name string,
	kind string,
	waitingMessage string,
	liveMessage string,
	observedAt []time.Time,
) FeedHealth {
	feed := FeedHealth{
		ID:          id,
		Name:        name,
		Kind:        kind,
		Status:      "stale",
		LastEventAt: now,
		Message:     waitingMessage,
	}
	latest := latestTime(observedAt)
	if latest.IsZero() {
		return feed
	}
	feed.LastEventAt = latest
	if now.Sub(latest) > freshness {
		feed.Message = "Last graph-backed state is outside freshness window"
		return feed
	}
	feed.Status = "live"
	feed.Message = liveMessage
	return feed
}

func filterTracksBySource(tracks []Track, source string) []Track {
	filtered := make([]Track, 0, len(tracks))
	for _, track := range tracks {
		if track.Source == source {
			filtered = append(filtered, track)
		}
	}
	return filtered
}

func trackObservationTimes(tracks []Track) []time.Time {
	times := make([]time.Time, 0, len(tracks))
	for _, track := range tracks {
		times = append(times, track.UpdatedAt)
	}
	return times
}

func taskObservationTimes(tasks []Task) []time.Time {
	times := make([]time.Time, 0, len(tasks))
	for _, task := range tasks {
		times = append(times, task.UpdatedAt)
	}
	return times
}

func advisoryObservationTimes(advisories []Advisory) []time.Time {
	times := make([]time.Time, 0, len(advisories))
	for _, advisory := range advisories {
		times = append(times, advisory.UpdatedAt)
	}
	return times
}

func hazardObservationTimes(hazards []Hazard) []time.Time {
	times := make([]time.Time, 0, len(hazards))
	for _, hazard := range hazards {
		times = append(times, hazard.UpdatedAt)
	}
	return times
}

func latestTime(times []time.Time) time.Time {
	var latest time.Time
	for _, next := range times {
		if next.After(latest) {
			latest = next
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

func trackFromEntity(entity graph.EntityState, now time.Time, freshness time.Duration) (Track, string, bool) {
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
	source := stringProperty(entity, copmodel.ProvenanceSource, sourceFromEntityID(entity.ID))
	return Track{
		ID:         entity.ID,
		Label:      trackLabel(entity),
		Source:     source,
		Status:     freshnessStatus(stringProperty(entity, copmodel.TrackStatus, "unknown"), now, updatedAt, freshness),
		Position:   position,
		Velocity:   stringProperty(entity, copmodel.TrackVelocity, ""),
		Confidence: confidence(entity),
		UpdatedAt:  updatedAt,
		Provenance: Provenance{
			Owner:     ownerForSource(source),
			SourceRef: stringProperty(entity, copmodel.ProvenanceSourceRef, ""),
			Observed:  updatedAt,
		},
	}, sourceID, true
}

func taskFromEntity(entity graph.EntityState, now time.Time, freshness time.Duration) (Task, bool) {
	updatedAt := observedAt(entity, copmodel.ProvenanceObservedAt)
	position := optionalPoint(entity, copmodel.TaskPosition)
	source := stringProperty(entity, copmodel.ProvenanceSource, sourceFromEntityID(entity.ID))
	return Task{
		ID:          entity.ID,
		Label:       stringProperty(entity, copmodel.TaskName, nativeOrInstanceLabel(entity, copmodel.TaskNativeID)),
		Kind:        stringProperty(entity, copmodel.TaskKind, "task"),
		Source:      source,
		Status:      freshnessStatus(stringProperty(entity, copmodel.TaskStatus, "unknown"), now, updatedAt, freshness),
		Position:    position,
		Description: stringProperty(entity, copmodel.TaskDescription, ""),
		Confidence:  confidence(entity),
		UpdatedAt:   updatedAt,
		Provenance: Provenance{
			Owner:     ownerForSource(source),
			SourceRef: stringProperty(entity, copmodel.ProvenanceSourceRef, ""),
			Observed:  updatedAt,
		},
	}, true
}

func advisoryFromEntity(entity graph.EntityState, now time.Time, freshness time.Duration) (Advisory, bool) {
	text := stringProperty(entity, copmodel.AdvisoryText, "")
	if text == "" {
		return Advisory{}, false
	}
	updatedAt := observedAt(entity, copmodel.ProvenanceObservedAt)
	position := optionalPoint(entity, copmodel.AdvisoryPosition)
	source := stringProperty(entity, copmodel.ProvenanceSource, sourceFromEntityID(entity.ID))
	sender := stringProperty(entity, copmodel.AdvisorySender, "")
	label := text
	if sender != "" {
		label = "GeoChat " + sender
	}
	return Advisory{
		ID:         entity.ID,
		Label:      label,
		Kind:       stringProperty(entity, copmodel.AdvisoryKind, "advisory"),
		Source:     source,
		Status:     freshnessStatus(stringProperty(entity, copmodel.AdvisoryStatus, "unknown"), now, updatedAt, freshness),
		Text:       text,
		Sender:     sender,
		Position:   position,
		Confidence: confidence(entity),
		UpdatedAt:  updatedAt,
		Provenance: Provenance{
			Owner:     ownerForSource(source),
			SourceRef: stringProperty(entity, copmodel.ProvenanceSourceRef, ""),
			Observed:  updatedAt,
		},
	}, true
}

func hazardFromEntity(entity graph.EntityState) (Hazard, bool) {
	evidence, ok := hazardEvidenceDocument(entity)
	if !ok {
		return Hazard{}, false
	}
	geometry := hazardGeometry(evidence)
	if len(geometry) < 3 {
		return Hazard{}, false
	}
	updatedAt := observedAt(entity, copmodel.ProvenanceObservedAt)
	source := stringProperty(
		entity,
		copmodel.HazardSource,
		stringProperty(entity, copmodel.ProvenanceSource, sourceFromEntityID(entity.ID)),
	)
	return Hazard{
		ID:         entity.ID,
		Label:      hazardLabel(entity, evidence),
		Kind:       hazardKind(evidence),
		Severity:   strings.ToLower(firstNonEmptyString(evidence.Severity, "unknown")),
		Geometry:   geometry,
		Source:     source,
		Confidence: confidence(entity),
		UpdatedAt:  updatedAt,
		Provenance: Provenance{
			Owner:     ownerForSource(source),
			SourceRef: stringProperty(entity, copmodel.ProvenanceSourceRef, ""),
			Observed:  updatedAt,
		},
	}, true
}

func hazardEvidenceDocument(entity graph.EntityState) (copmodel.HazardEvidenceDocument, bool) {
	value, ok := propertyValue(entity, copmodel.HazardEvidence)
	if !ok {
		return copmodel.HazardEvidenceDocument{}, false
	}
	var doc copmodel.HazardEvidenceDocument
	if text, ok := stringFromAny(value); ok {
		if err := json.Unmarshal([]byte(text), &doc); err != nil {
			return copmodel.HazardEvidenceDocument{}, false
		}
	} else {
		data, err := json.Marshal(value)
		if err != nil {
			return copmodel.HazardEvidenceDocument{}, false
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return copmodel.HazardEvidenceDocument{}, false
		}
	}
	if doc.Identifier == "" && doc.Event == "" && len(doc.Polygons) == 0 && len(doc.Circles) == 0 {
		return copmodel.HazardEvidenceDocument{}, false
	}
	return doc, true
}

func hazardLabel(entity graph.EntityState, evidence copmodel.HazardEvidenceDocument) string {
	label := firstNonEmptyString(evidence.Event, stringProperty(entity, copmodel.HazardAdvisoryText, ""), instanceLabel(entity.ID))
	if evidence.AreaDesc == "" {
		return label
	}
	if strings.Contains(strings.ToLower(label), strings.ToLower(evidence.AreaDesc)) {
		return label
	}
	return label + ": " + evidence.AreaDesc
}

func hazardKind(evidence copmodel.HazardEvidenceDocument) string {
	event := strings.TrimSpace(evidence.Event)
	if event == "" {
		return "cap-alert"
	}
	return "cap-" + strings.ReplaceAll(strings.ToLower(event), " ", "-")
}

func hazardGeometry(evidence copmodel.HazardEvidenceDocument) []GeoPoint {
	for _, polygon := range evidence.Polygons {
		points := make([]GeoPoint, 0, len(polygon))
		for _, point := range polygon {
			next, ok := hazardEvidencePoint(point)
			if ok {
				points = append(points, next)
			}
		}
		if len(points) >= 3 {
			return points
		}
	}
	for _, circle := range evidence.Circles {
		points := hazardCircleGeometry(circle, 24)
		if len(points) >= 3 {
			return points
		}
	}
	return nil
}

func hazardEvidencePoint(point copmodel.HazardEvidencePoint) (GeoPoint, bool) {
	if point.Lat < -90 || point.Lat > 90 || point.Lon < -180 || point.Lon > 180 {
		return GeoPoint{}, false
	}
	return GeoPoint{Lat: point.Lat, Lon: point.Lon}, true
}

func hazardCircleGeometry(circle copmodel.HazardEvidenceCircle, segments int) []GeoPoint {
	if circle.RadiusKM <= 0 || segments < 3 {
		return nil
	}
	center, ok := hazardEvidencePoint(circle.Center)
	if !ok {
		return nil
	}
	const kmPerLatDegree = 111.32
	latDelta := circle.RadiusKM / kmPerLatDegree
	lonScale := kmPerLatDegree * math.Cos(center.Lat*math.Pi/180)
	if math.Abs(lonScale) < 0.001 {
		return nil
	}
	lonDelta := circle.RadiusKM / lonScale
	points := make([]GeoPoint, 0, segments)
	for i := 0; i < segments; i++ {
		radians := 2 * math.Pi * float64(i) / float64(segments)
		points = append(points, GeoPoint{
			Lat: center.Lat + latDelta*math.Sin(radians),
			Lon: center.Lon + lonDelta*math.Cos(radians),
		})
	}
	return points
}

func optionalPoint(entity graph.EntityState, predicate string) *GeoPoint {
	value, ok := propertyValue(entity, predicate)
	if !ok {
		return nil
	}
	point, ok := geoPointFromWKT(value)
	if !ok {
		return nil
	}
	return &point
}

func freshnessStatus(status string, now time.Time, updatedAt time.Time, freshness time.Duration) string {
	if status == "" {
		status = "unknown"
	}
	if freshness <= 0 || updatedAt.IsZero() || now.Sub(updatedAt) <= freshness || strings.HasPrefix(status, "stale") {
		return status
	}
	if _, suffix, ok := strings.Cut(status, "."); ok && suffix != "" {
		return "stale." + suffix
	}
	return "stale"
}

func sourceFromEntityID(entityID string) string {
	if strings.Contains(entityID, ".cop.tak.") {
		return "tak-cot"
	}
	if strings.Contains(entityID, ".cop.mavlink.") {
		return "mavlink"
	}
	if strings.Contains(entityID, ".cop.cap.") {
		return "cap"
	}
	return "graph"
}

func ownerForSource(source string) string {
	switch source {
	case "tak-cot":
		return copmodel.OwnerTAK
	case "mavlink":
		return copmodel.OwnerMAVLink
	case "cap":
		return copmodel.OwnerCAP
	default:
		return copmodel.OwnerFusion
	}
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
	if uid := cotUIDFromNativeID(nativeID); uid != "" {
		return uid
	}
	if systemID := systemIDFromNativeID(nativeID); systemID != "" {
		return "UAS " + systemID
	}
	if systemID := systemIDFromEntityID(entity.ID); systemID != "" {
		return "UAS " + systemID
	}
	return instanceLabel(entity.ID)
}

func nativeOrInstanceLabel(entity graph.EntityState, predicate string) string {
	if uid := cotUIDFromNativeID(stringProperty(entity, predicate, "")); uid != "" {
		return uid
	}
	return instanceLabel(entity.ID)
}

func cotUIDFromNativeID(nativeID string) string {
	nativeID = strings.TrimSpace(nativeID)
	if !strings.HasPrefix(nativeID, "cot.uid.") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(nativeID, "cot.uid."))
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

func normalizeCoTUIDs(uids []CoTUIDRef) []CoTUIDRef {
	out := make([]CoTUIDRef, 0, len(uids))
	for _, uid := range uids {
		uid.UID = strings.TrimSpace(uid.UID)
		if uid.UID == "" {
			continue
		}
		if uid.Org == "" {
			uid.Org = "c360"
		}
		if uid.Platform == "" {
			uid.Platform = "edge"
		}
		out = append(out, uid)
	}
	return out
}

func normalizeCAPAlerts(alerts []CAPAlertRef) []CAPAlertRef {
	out := make([]CAPAlertRef, 0, len(alerts))
	for _, alert := range alerts {
		alert.Identifier = strings.TrimSpace(alert.Identifier)
		if alert.Identifier == "" {
			continue
		}
		if alert.Org == "" {
			alert.Org = "c360"
		}
		if alert.Platform == "" {
			alert.Platform = "edge"
		}
		out = append(out, alert)
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

func (s CoTUIDRef) assetID() string {
	return cotprojector.EntityID(s.Org, s.Platform, copmodel.EntityAsset, s.UID)
}

func (s CoTUIDRef) trackID() string {
	return cotprojector.EntityID(s.Org, s.Platform, copmodel.EntityTrack, s.UID)
}

func (s CoTUIDRef) taskID() string {
	return cotprojector.EntityID(s.Org, s.Platform, copmodel.EntityTask, s.UID)
}

func (s CoTUIDRef) advisoryID() string {
	return cotprojector.EntityID(s.Org, s.Platform, copmodel.EntityAdvisory, s.UID)
}

func (s CAPAlertRef) hazardID() string {
	return capprojector.EntityID(s.Org, s.Platform, s.Identifier)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
