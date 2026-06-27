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
	SubjectGraphQueryPrefix      = "graph.query.prefix"
	DefaultGraphQueryTimeout     = 2 * time.Second
	DefaultGraphDiscoveryLimit   = 500
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
	prefixSubject  string
	queryTimeout   time.Duration
	now            func() time.Time
	fallback       SnapshotProvider
	mavlinkSystems []MAVLinkSystemRef
	cotUIDs        []CoTUIDRef
	capAlerts      []CAPAlertRef
	discovery      GraphDiscoveryConfig
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

type GraphDiscoveryConfig struct {
	Enabled bool
	Limit   int
	Scopes  []GraphDiscoveryScope
}

type GraphDiscoveryScope struct {
	Org      string
	Platform string
}

type GraphProviderOption func(*GraphProvider)

func NewGraphProvider(requester GraphRequester, opts ...GraphProviderOption) (*GraphProvider, error) {
	if requester == nil {
		return nil, fmt.Errorf("graph snapshot provider requires a requester")
	}
	provider := &GraphProvider{
		requester:     requester,
		querySubject:  SubjectGraphQueryEntity,
		prefixSubject: SubjectGraphQueryPrefix,
		queryTimeout:  DefaultGraphQueryTimeout,
		now:           time.Now,
		freshness:     DefaultFeedFreshnessWindow,
		discovery: GraphDiscoveryConfig{
			Enabled: true,
			Limit:   DefaultGraphDiscoveryLimit,
			Scopes: []GraphDiscoveryScope{{
				Org:      "c360",
				Platform: "edge",
			}},
		},
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
	provider.discovery = normalizeGraphDiscovery(provider.discovery)
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
	if provider.prefixSubject == "" {
		provider.prefixSubject = SubjectGraphQueryPrefix
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

func WithGraphDiscovery(enabled bool) GraphProviderOption {
	return func(provider *GraphProvider) {
		provider.discovery.Enabled = enabled
	}
}

func WithGraphDiscoveryLimit(limit int) GraphProviderOption {
	return func(provider *GraphProvider) {
		if limit > 0 {
			provider.discovery.Limit = limit
		}
	}
}

func WithGraphDiscoveryScopes(scopes []GraphDiscoveryScope) GraphProviderOption {
	return func(provider *GraphProvider) {
		provider.discovery.Scopes = append([]GraphDiscoveryScope(nil), scopes...)
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
	footprintsByID := make(map[string]graph.EntityState)
	weatherByID := make(map[string]graph.EntityState)
	associationsByID := make(map[string]graph.EntityState)
	associationReviewsByID := make(map[string]graph.EntityState)
	var firstErr error

	var discovered graphDiscoveryResult
	if p.discovery.Enabled {
		if result, err := p.discoverInto(
			ctx,
			assetsByID,
			tracksByID,
			tasksByID,
			advisoriesByID,
			hazardsByID,
			footprintsByID,
			weatherByID,
			associationsByID,
			associationReviewsByID,
		); err != nil {
			firstErr = errors.Join(firstErr, err)
			discovered = result
		} else {
			discovered = result
		}
	}

	if !discovered.mavlink {
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
	}

	if !discovered.cot {
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
	}

	if !discovered.cap {
		for _, alert := range p.capAlerts {
			if hazard, ok, err := p.queryEntity(ctx, alert.hazardID()); err != nil {
				firstErr = errors.Join(firstErr, err)
			} else if ok {
				hazardsByID[hazard.ID] = hazard
			}
		}
	}

	if len(assetsByID) == 0 &&
		len(tracksByID) == 0 &&
		len(tasksByID) == 0 &&
		len(advisoriesByID) == 0 &&
		len(hazardsByID) == 0 &&
		len(footprintsByID) == 0 &&
		len(weatherByID) == 0 &&
		len(associationsByID) == 0 &&
		len(associationReviewsByID) == 0 {
		if p.fallback != nil {
			return p.fallback.Snapshot(ctx)
		}
		if firstErr != nil {
			return Snapshot{}, firstErr
		}
	}

	return p.snapshotFromGraph(
		assetsByID,
		tracksByID,
		tasksByID,
		advisoriesByID,
		hazardsByID,
		footprintsByID,
		weatherByID,
		associationsByID,
		associationReviewsByID,
		discovered.diagnostics,
	), nil
}

type graphDiscoveryTarget struct {
	Prefix   string
	Org      string
	Platform string
	Source   string
	Kind     string
	Family   string
}

type graphDiscoveryResult struct {
	mavlink     bool
	cot         bool
	command     bool
	cap         bool
	sapient     bool
	klv         bool
	weather     bool
	fusion      bool
	diagnostics []DiscoveryDiagnostic
}

func (p *GraphProvider) discoverInto(
	ctx context.Context,
	assetsByID map[string]graph.EntityState,
	tracksByID map[string]graph.EntityState,
	tasksByID map[string]graph.EntityState,
	advisoriesByID map[string]graph.EntityState,
	hazardsByID map[string]graph.EntityState,
	footprintsByID map[string]graph.EntityState,
	weatherByID map[string]graph.EntityState,
	associationsByID map[string]graph.EntityState,
	associationReviewsByID map[string]graph.EntityState,
) (graphDiscoveryResult, error) {
	var result graphDiscoveryResult
	var firstErr error
	for _, target := range p.discoveryTargets() {
		entities, truncated, err := p.queryPrefix(ctx, target.Prefix)
		if err != nil {
			firstErr = errors.Join(firstErr, err)
			result.diagnostics = append(result.diagnostics, target.diagnostic(p.discovery.Limit, 0, false, err))
			continue
		}
		var count int
		for _, entity := range entities {
			if entity.ID == "" {
				continue
			}
			count++
			switch target.Family {
			case "mavlink":
				result.mavlink = true
			case "cot":
				result.cot = true
			case "command":
				result.command = true
			case "cap":
				result.cap = true
			case "klv":
				result.klv = true
			case "sapient":
				result.sapient = true
			case "weather":
				result.weather = true
			case "fusion":
				result.fusion = true
			}
			switch target.Kind {
			case copmodel.EntityAsset:
				assetsByID[entity.ID] = entity
			case copmodel.EntityTrack:
				tracksByID[entity.ID] = entity
			case copmodel.EntityTask:
				tasksByID[entity.ID] = entity
			case copmodel.EntityAdvisory:
				advisoriesByID[entity.ID] = entity
			case copmodel.EntityHazardArea:
				hazardsByID[entity.ID] = entity
			case copmodel.EntitySensorFootprint:
				footprintsByID[entity.ID] = entity
			case copmodel.EntityWeatherObservation:
				weatherByID[entity.ID] = entity
			case copmodel.EntityAssociation:
				associationsByID[entity.ID] = entity
			case copmodel.EntityAssociationReview:
				associationReviewsByID[entity.ID] = entity
			}
		}
		result.diagnostics = append(result.diagnostics, target.diagnostic(p.discovery.Limit, count, truncated, nil))
	}
	return result, firstErr
}

func (p *GraphProvider) discoveryTargets() []graphDiscoveryTarget {
	targets := make([]graphDiscoveryTarget, 0, len(p.discovery.Scopes)*14)
	for _, scope := range p.discovery.Scopes {
		targets = append(targets,
			newGraphDiscoveryTarget(scope, "mavlink", copmodel.EntityAsset, "mavlink"),
			newGraphDiscoveryTarget(scope, "mavlink", copmodel.EntityTrack, "mavlink"),
			newGraphDiscoveryTarget(scope, "mavlink", copmodel.EntityTask, "mavlink"),
			newGraphDiscoveryTarget(scope, "tak", copmodel.EntityAsset, "cot"),
			newGraphDiscoveryTarget(scope, "tak", copmodel.EntityTrack, "cot"),
			newGraphDiscoveryTarget(scope, "tak", copmodel.EntityTask, "cot"),
			newGraphDiscoveryTarget(scope, "tak", copmodel.EntityAdvisory, "cot"),
			newGraphDiscoveryTarget(scope, "command", copmodel.EntityTask, "command"),
			newGraphDiscoveryTarget(scope, "cap", copmodel.EntityHazardArea, "cap"),
			newGraphDiscoveryTarget(scope, "adsb", copmodel.EntityTrack, "adsb"),
			newGraphDiscoveryTarget(scope, "sapient", copmodel.EntityTrack, "sapient"),
			newGraphDiscoveryTarget(scope, "klv", copmodel.EntitySensorFootprint, "klv"),
			newGraphDiscoveryTarget(scope, "weather", copmodel.EntityWeatherObservation, "weather"),
			newGraphDiscoveryTarget(scope, "fusion", copmodel.EntityAssociation, "fusion"),
			newGraphDiscoveryTarget(scope, "fusion", copmodel.EntityAssociationReview, "fusion"),
		)
	}
	return targets
}

func newGraphDiscoveryTarget(scope GraphDiscoveryScope, source string, entityType string, family string) graphDiscoveryTarget {
	return graphDiscoveryTarget{
		Prefix:   graphEntityPrefix(scope.Org, scope.Platform, source, entityType),
		Org:      scope.Org,
		Platform: scope.Platform,
		Source:   source,
		Kind:     entityType,
		Family:   family,
	}
}

func (t graphDiscoveryTarget) diagnostic(limit int, count int, truncated bool, err error) DiscoveryDiagnostic {
	diagnostic := DiscoveryDiagnostic{
		Org:        t.Org,
		Platform:   t.Platform,
		Source:     t.Source,
		Family:     t.Family,
		EntityType: t.Kind,
		Prefix:     t.Prefix,
		Count:      count,
		Limit:      limit,
		AtLimit:    truncated,
	}
	if err != nil {
		diagnostic.Error = err.Error()
	}
	return diagnostic
}

func (p *GraphProvider) queryPrefix(ctx context.Context, prefix string) ([]graph.EntityState, bool, error) {
	limit := p.discovery.Limit
	if limit <= 0 {
		limit = DefaultGraphDiscoveryLimit
	}
	capacity := limit
	if capacity > 64 {
		capacity = 64
	}
	entities := make([]graph.EntityState, 0, capacity)
	remaining := limit
	cursor := ""
	for {
		pageLimit := prefixPageLimit(remaining)
		if pageLimit <= 0 {
			return entities, cursor != "", nil
		}
		page, err := p.queryPrefixPage(ctx, prefix, cursor, pageLimit)
		if err != nil {
			return nil, false, err
		}
		pageEntities := page.Entities
		if len(pageEntities) > remaining {
			entities = append(entities, pageEntities[:remaining]...)
			return entities, true, nil
		}
		entities = append(entities, pageEntities...)
		remaining -= len(pageEntities)
		if page.NextCursor == "" {
			return entities, false, nil
		}
		if remaining <= 0 {
			return entities, true, nil
		}
		if len(pageEntities) == 0 {
			return nil, false, fmt.Errorf("query graph prefix %s: empty page returned continuation cursor", prefix)
		}
		if page.NextCursor == cursor {
			return nil, false, fmt.Errorf("query graph prefix %s: repeated continuation cursor %q", prefix, cursor)
		}
		cursor = page.NextCursor
	}
}

func prefixPageLimit(remaining int) int {
	if remaining <= 0 {
		return 0
	}
	if remaining > graph.MaxPrefixQueryLimit {
		return graph.MaxPrefixQueryLimit
	}
	return remaining
}

func (p *GraphProvider) queryPrefixPage(
	ctx context.Context,
	prefix string,
	cursor string,
	limit int,
) (graph.PrefixQueryResponse, error) {
	body, err := json.Marshal(graph.PrefixQueryRequest{
		Prefix: prefix,
		Limit:  limit,
		Cursor: cursor,
	})
	if err != nil {
		return graph.PrefixQueryResponse{}, err
	}
	response, err := p.requestPrefix(ctx, body)
	if err != nil {
		return graph.PrefixQueryResponse{}, fmt.Errorf("query graph prefix %s: %w", prefix, err)
	}
	if isLegacyErrorResponse(response) {
		if isNotFoundText(string(response)) {
			return graph.PrefixQueryResponse{}, nil
		}
		return graph.PrefixQueryResponse{}, fmt.Errorf("query graph prefix %s: %s", prefix, string(response))
	}
	var envelope struct {
		Entities   []graph.EntityState `json:"entities"`
		NextCursor string              `json:"next_cursor,omitempty"`
		Data       struct {
			Entities   []graph.EntityState `json:"entities"`
			NextCursor string              `json:"next_cursor,omitempty"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(response, &envelope); err != nil {
		return graph.PrefixQueryResponse{}, fmt.Errorf("decode graph prefix %s: %w", prefix, err)
	}
	if envelope.Error != "" {
		return graph.PrefixQueryResponse{}, fmt.Errorf("query graph prefix %s: %s", prefix, envelope.Error)
	}
	result := graph.PrefixQueryResponse{
		Entities:   envelope.Entities,
		NextCursor: envelope.NextCursor,
	}
	if len(result.Entities) == 0 && result.NextCursor == "" &&
		(len(envelope.Data.Entities) > 0 || envelope.Data.NextCursor != "") {
		result.Entities = envelope.Data.Entities
		result.NextCursor = envelope.Data.NextCursor
	}
	return result, nil
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

func (p *GraphProvider) requestPrefix(ctx context.Context, body []byte) ([]byte, error) {
	if classified, ok := p.requester.(classifiedGraphRequester); ok {
		return classified.RequestClassified(ctx, p.prefixSubject, body, p.queryTimeout)
	}
	return p.requester.Request(ctx, p.prefixSubject, body, p.queryTimeout)
}

func (p *GraphProvider) snapshotFromGraph(
	assetsByID map[string]graph.EntityState,
	tracksByID map[string]graph.EntityState,
	tasksByID map[string]graph.EntityState,
	advisoriesByID map[string]graph.EntityState,
	hazardsByID map[string]graph.EntityState,
	footprintsByID map[string]graph.EntityState,
	weatherByID map[string]graph.EntityState,
	associationsByID map[string]graph.EntityState,
	associationReviewsByID map[string]graph.EntityState,
	discoveryDiagnostics []DiscoveryDiagnostic,
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
		hazard, ok := hazardFromEntity(entity, now, p.freshness)
		if ok {
			hazards = append(hazards, hazard)
		}
	}

	footprints := make([]SensorFootprint, 0, len(footprintsByID))
	for _, entity := range sortedEntities(footprintsByID) {
		footprint, ok := sensorFootprintFromEntity(entity, now, p.freshness)
		if ok {
			footprints = append(footprints, footprint)
		}
	}

	weather := make([]WeatherObservation, 0, len(weatherByID))
	for _, entity := range sortedEntities(weatherByID) {
		observation, ok := weatherObservationFromEntity(entity, now, p.freshness)
		if ok {
			weather = append(weather, observation)
		}
	}

	associations := make([]Association, 0, len(associationsByID))
	associationReviews := associationReviewsByAssociationID(associationReviewsByID)
	for _, entity := range sortedEntities(associationsByID) {
		association, ok := associationFromEntity(entity, now, p.freshness)
		if ok {
			if review, reviewed := associationReviews[association.ID]; reviewed {
				association.OperatorReview = &review
			}
			associations = append(associations, association)
		}
	}

	feeds := firstPhaseFeedHealth(now, p.freshness, tracks, tasks, advisories, hazards, footprints, weather, associations)
	discoveryDiagnostics = normalizeDiscoveryDiagnostics(discoveryDiagnostics)
	alerts := diagnosticAlertsFromDiscovery(discoveryDiagnostics, now)
	return Snapshot{
		GeneratedAt: now,
		Scenario:    "phase-1-live-graph",
		Summary: Summary{
			ActiveTracks:           len(tracks),
			ActiveTasks:            len(tasks),
			ActiveAdvisories:       len(advisories),
			ActiveSensorFootprints: len(footprints),
			ActiveWeather:          len(weather),
			ActiveAssociations:     len(associations),
			ActiveAlerts:           len(alerts),
			StaleFeeds:             countFeeds(feeds, "stale"),
		},
		Diagnostics: SnapshotDiagnostics{
			Discovery: discoveryDiagnostics,
		},
		Feeds:            feeds,
		Assets:           assets,
		Tracks:           tracks,
		Tasks:            tasks,
		Advisories:       advisories,
		Hazards:          hazards,
		SensorFootprints: footprints,
		Weather:          weather,
		Associations:     associations,
		Alerts:           alerts,
	}
}

func normalizeDiscoveryDiagnostics(diagnostics []DiscoveryDiagnostic) []DiscoveryDiagnostic {
	if diagnostics == nil {
		return []DiscoveryDiagnostic{}
	}
	return diagnostics
}

func diagnosticAlertsFromDiscovery(diagnostics []DiscoveryDiagnostic, now time.Time) []Alert {
	alerts := make([]Alert, 0)
	for _, diagnostic := range diagnostics {
		if diagnostic.Error != "" {
			alerts = append(alerts, Alert{
				ID:        discoveryAlertID(diagnostic, "error"),
				Label:     fmt.Sprintf("%s %s discovery error", sourceLabel(diagnostic.Source), entityTypeLabel(diagnostic.EntityType)),
				Severity:  "warning",
				Status:    "active",
				EntityID:  feedIDForDiscoverySource(diagnostic.Source),
				Reason:    fmt.Sprintf("Prefix query for %s returned an error: %s", diagnostic.Prefix, diagnostic.Error),
				UpdatedAt: now,
			})
		}
		if diagnostic.AtLimit {
			alerts = append(alerts, Alert{
				ID:        discoveryAlertID(diagnostic, "limit"),
				Label:     fmt.Sprintf("%s %s discovery truncated", sourceLabel(diagnostic.Source), entityTypeLabel(diagnostic.EntityType)),
				Severity:  "warning",
				Status:    "active",
				EntityID:  feedIDForDiscoverySource(diagnostic.Source),
				Reason:    fmt.Sprintf("Prefix query returned %d entities and SemStreams reported more continuation state after the configured limit of %d; snapshot results may be truncated.", diagnostic.Count, diagnostic.Limit),
				UpdatedAt: now,
			})
		}
	}
	return alerts
}

func discoveryAlertID(diagnostic DiscoveryDiagnostic, suffix string) string {
	return strings.Join([]string{
		"alert",
		"discovery",
		alertIDSegment(diagnostic.Org),
		alertIDSegment(diagnostic.Platform),
		alertIDSegment(diagnostic.Source),
		alertIDSegment(diagnostic.EntityType),
		alertIDSegment(suffix),
	}, ".")
}

func alertIDSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "unknown"
	}
	return value
}

func sourceLabel(source string) string {
	switch source {
	case "mavlink":
		return "MAVLink"
	case "tak":
		return "TAK/CoT"
	case "command":
		return "Command"
	case "cap":
		return "CAP"
	case "adsb":
		return "ADS-B"
	case "sapient":
		return "SAPIENT"
	case "klv":
		return "KLV"
	case "weather":
		return "Weather"
	case "fusion":
		return "Fusion"
	default:
		return source
	}
}

func entityTypeLabel(entityType string) string {
	return strings.ReplaceAll(entityType, "_", " ")
}

func feedIDForDiscoverySource(source string) string {
	switch source {
	case "mavlink":
		return "feed.mavlink"
	case "tak":
		return "feed.tak"
	case "command":
		return "feed.command"
	case "cap":
		return "feed.cap"
	case "adsb":
		return "feed.adsb"
	case "sapient":
		return "feed.sapient"
	case "klv":
		return "feed.klv"
	case "weather":
		return "feed.weather"
	case "fusion":
		return "feed.fusion"
	default:
		return "feed." + alertIDSegment(source)
	}
}

func firstPhaseFeedHealth(
	now time.Time,
	freshness time.Duration,
	tracks []Track,
	tasks []Task,
	advisories []Advisory,
	hazards []Hazard,
	footprints []SensorFootprint,
	weather []WeatherObservation,
	associations []Association,
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
	takTasks := filterTasksBySource(tasks, "tak-cot")
	takObservations := append(
		trackObservationTimes(filterTracksBySource(tracks, "tak-cot")),
		taskObservationTimes(takTasks)...,
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
	commandTasks := filterTasksBySource(tasks, "command")
	command := feedHealthFromObservations(
		now,
		freshness,
		"feed.command",
		"Command",
		"control",
		"Command lifecycle replay gate pending",
		"Graph-backed command-intent lifecycle state",
		taskObservationTimes(commandTasks),
	)
	if len(commandTasks) == 0 {
		command.Status = "planned"
		command.LastEventAt = now.Add(-41 * time.Minute)
	}
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
	adsbTracks := filterTracksBySource(tracks, "adsb")
	adsb := feedHealthFromObservations(
		now,
		freshness,
		"feed.adsb",
		"ADS-B",
		"air-picture",
		"ADS-B projection gate pending",
		"Graph-backed ADS-B aircraft tracks",
		trackObservationTimes(adsbTracks),
	)
	if len(adsbTracks) == 0 {
		adsb.Status = "planned"
		adsb.LastEventAt = now.Add(-45 * time.Minute)
	}
	sapientTracks := filterTracksBySource(tracks, "sapient")
	sapient := feedHealthFromObservations(
		now,
		freshness,
		"feed.sapient",
		"SAPIENT",
		"sensor-network",
		"SAPIENT projection gate pending",
		"Graph-backed SAPIENT absolute-location detections",
		trackObservationTimes(sapientTracks),
	)
	if len(sapientTracks) == 0 {
		sapient.Status = "planned"
		sapient.LastEventAt = now.Add(-47 * time.Minute)
	}
	klv := feedHealthFromObservations(
		now,
		freshness,
		"feed.klv",
		"KLV",
		"sensor-footprint",
		"KLV sensor/footprint graph proof pending",
		"Graph-backed KLV sensor/footprint state",
		sensorFootprintObservationTimes(footprints),
	)
	if len(footprints) == 0 {
		klv.Status = "planned"
		klv.LastEventAt = now.Add(-50 * time.Minute)
	}
	weatherFeed := weatherFeedHealth(now, freshness, weather)
	fusion := feedHealthFromObservations(
		now,
		freshness,
		"feed.fusion",
		"Fusion",
		"inference",
		"Fusion association graph proof pending",
		"Graph-backed fusion association evidence",
		associationObservationTimes(associations),
	)
	if len(associations) == 0 {
		fusion.Status = "planned"
		fusion.LastEventAt = now.Add(-54 * time.Minute)
	}
	return []FeedHealth{mavlink, tak, command, cap, adsb, sapient, klv, weatherFeed, fusion}
}

func weatherFeedHealth(now time.Time, freshness time.Duration, observations []WeatherObservation) FeedHealth {
	feed := feedHealthFromObservations(
		now,
		freshness,
		"feed.weather",
		"Weather",
		"tactical-weather",
		"Weather graph proof pending",
		"Graph-backed tactical weather observations",
		weatherObservationTimes(observations),
	)
	if len(observations) == 0 {
		feed.Status = "planned"
		feed.LastEventAt = now.Add(-52 * time.Minute)
		return feed
	}
	for _, observation := range observations {
		if observation.Status == "fresh" {
			feed.Status = "live"
			feed.Message = "Graph-backed tactical weather observations"
			return feed
		}
	}
	return feed
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

func filterTasksBySource(tasks []Task, source string) []Task {
	filtered := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Source == source {
			filtered = append(filtered, task)
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

func sensorFootprintObservationTimes(footprints []SensorFootprint) []time.Time {
	times := make([]time.Time, 0, len(footprints))
	for _, footprint := range footprints {
		times = append(times, footprint.UpdatedAt)
	}
	return times
}

func weatherObservationTimes(observations []WeatherObservation) []time.Time {
	times := make([]time.Time, 0, len(observations))
	for _, observation := range observations {
		times = append(times, observation.UpdatedAt)
	}
	return times
}

func associationObservationTimes(associations []Association) []time.Time {
	times := make([]time.Time, 0, len(associations))
	for _, association := range associations {
		times = append(times, association.UpdatedAt)
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
	updatedAt := latestObservedAt(entity, copmodel.ProvenanceObservedAt)
	position := optionalPoint(entity, copmodel.TaskPosition)
	entitySource := sourceFromEntityID(entity.ID)
	source := latestStringProperty(entity, copmodel.ProvenanceSource, entitySource)
	if entitySource == "command" {
		source = "command"
	}
	expiresAt := optionalLatestTime(entity, copmodel.TaskExpiresAt)
	return Task{
		ID:                  entity.ID,
		Label:               latestStringProperty(entity, copmodel.TaskName, nativeOrInstanceLabel(entity, copmodel.TaskNativeID)),
		Kind:                latestStringProperty(entity, copmodel.TaskKind, "task"),
		Source:              source,
		Status:              freshnessStatus(latestStringProperty(entity, copmodel.TaskStatus, "unknown"), now, updatedAt, freshness),
		Position:            position,
		Description:         latestStringProperty(entity, copmodel.TaskDescription, ""),
		TargetID:            latestStringProperty(entity, copmodel.TaskTarget, ""),
		Authority:           latestStringProperty(entity, copmodel.TaskAuthority, ""),
		Priority:            optionalLatestInt(entity, copmodel.TaskPriority),
		ExpiresAt:           expiresAt,
		RequestedBy:         latestStringProperty(entity, copmodel.TaskRequestedBy, ""),
		CorrelationID:       latestStringProperty(entity, copmodel.TaskCorrelation, ""),
		DesiredState:        latestStringProperty(entity, copmodel.TaskDesired, ""),
		LocalOverridePolicy: latestStringProperty(entity, copmodel.TaskLocalOverridePolicy, ""),
		Confidence:          confidence(entity),
		UpdatedAt:           updatedAt,
		Provenance: Provenance{
			Owner:     ownerForSource(source),
			SourceRef: latestStringProperty(entity, copmodel.ProvenanceSourceRef, ""),
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

func hazardFromEntity(entity graph.EntityState, now time.Time, freshness time.Duration) (Hazard, bool) {
	evidence, ok := hazardEvidenceDocument(entity)
	if !ok {
		return Hazard{}, false
	}
	geometry := hazardGeometry(evidence)
	if len(geometry) < 3 {
		return Hazard{}, false
	}
	updatedAt := latestObservedAt(entity, copmodel.ProvenanceObservedAt)
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
		Status:     hazardStatus(evidence, now, updatedAt, freshness),
		Geometry:   geometry,
		Source:     source,
		Confidence: confidence(entity),
		UpdatedAt:  updatedAt,
		Provenance: Provenance{
			Owner:     ownerForSource(source),
			SourceRef: latestStringProperty(entity, copmodel.ProvenanceSourceRef, ""),
			Observed:  updatedAt,
		},
	}, true
}

func sensorFootprintFromEntity(entity graph.EntityState, now time.Time, freshness time.Duration) (SensorFootprint, bool) {
	sensorPosition, ok := latestPoint(entity, copmodel.SensorFootprintSensorPosition)
	if !ok {
		return SensorFootprint{}, false
	}
	frameCenter, ok := latestPoint(entity, copmodel.SensorFootprintFrameCenter)
	if !ok {
		return SensorFootprint{}, false
	}
	updatedAt := latestObservedAt(entity, copmodel.SensorFootprintObservedAt, copmodel.ProvenanceObservedAt)
	source := stringProperty(entity, copmodel.SensorFootprintSource, stringProperty(entity, copmodel.ProvenanceSource, "klv"))
	mediaRef := latestStringProperty(entity, copmodel.SensorFootprintMediaRef, "")
	packetRef := latestStringProperty(entity, copmodel.SensorFootprintPacketRef, "")
	platformDesignation := latestStringProperty(entity, copmodel.SensorFootprintPlatformDesignation, "")
	footprint := latestPolygon(entity, copmodel.SensorFootprintGeometry)
	return SensorFootprint{
		ID:                         entity.ID,
		Label:                      sensorFootprintLabel(entity, platformDesignation),
		Source:                     source,
		Status:                     freshnessStatus(sensorFootprintStatus(footprint), now, updatedAt, freshness),
		SensorPosition:             sensorPosition,
		FrameCenter:                frameCenter,
		Ray:                        []GeoPoint{sensorPosition, frameCenter},
		Footprint:                  footprint,
		SensorAltitudeMeters:       optionalFloat(entity, copmodel.SensorFootprintSensorAltitude),
		SensorAzimuthDegrees:       optionalFloat(entity, copmodel.SensorFootprintSensorAzimuth),
		SensorElevationDegrees:     optionalFloat(entity, copmodel.SensorFootprintSensorElevation),
		FrameCenterElevationMeters: optionalFloat(entity, copmodel.SensorFootprintFrameCenterElevation),
		MediaRef:                   mediaRef,
		PacketRef:                  packetRef,
		FrameTime:                  updatedAt,
		PlatformDesignation:        platformDesignation,
		ClaimPosture:               sensorFootprintClaimPosture(footprint),
		DecodedFields:              sensorFootprintDecodedFields(entity),
		Warnings:                   sensorFootprintWarnings(entity, mediaRef, packetRef),
		Confidence:                 confidence(entity),
		UpdatedAt:                  updatedAt,
		Provenance: Provenance{
			Owner:     ownerForSource(source),
			SourceRef: latestStringProperty(entity, copmodel.ProvenanceSourceRef, packetRef),
			Observed:  updatedAt,
		},
	}, true
}

func sensorFootprintStatus(footprint []GeoPoint) string {
	if len(footprint) >= 3 {
		return "active.footprint-polygon"
	}
	return "active.sensor-frame-center"
}

func sensorFootprintClaimPosture(footprint []GeoPoint) string {
	if len(footprint) >= 3 {
		return "sensor/frame-center and footprint polygon graph readback; tested MISB ST 0601 offset-corner subset; no STANAG conformance"
	}
	return "sensor-frame-center graph readback; no footprint polygon; no STANAG conformance"
}

func sensorFootprintLabel(entity graph.EntityState, platformDesignation string) string {
	if strings.TrimSpace(platformDesignation) != "" {
		return platformDesignation + " sensor footprint"
	}
	return "KLV " + instanceLabel(entity.ID)
}

func sensorFootprintDecodedFields(entity graph.EntityState) []string {
	fields := make([]string, 0, 10)
	for _, field := range []struct {
		name      string
		predicate string
	}{
		{name: "media_ref", predicate: copmodel.SensorFootprintMediaRef},
		{name: "packet_ref", predicate: copmodel.SensorFootprintPacketRef},
		{name: "observed_at", predicate: copmodel.SensorFootprintObservedAt},
		{name: "platform_designation", predicate: copmodel.SensorFootprintPlatformDesignation},
		{name: "sensor_position", predicate: copmodel.SensorFootprintSensorPosition},
		{name: "sensor_altitude_meters", predicate: copmodel.SensorFootprintSensorAltitude},
		{name: "sensor_azimuth_degrees", predicate: copmodel.SensorFootprintSensorAzimuth},
		{name: "sensor_elevation_degrees", predicate: copmodel.SensorFootprintSensorElevation},
		{name: "frame_center", predicate: copmodel.SensorFootprintFrameCenter},
		{name: "frame_center_elevation_meters", predicate: copmodel.SensorFootprintFrameCenterElevation},
		{name: "footprint_polygon", predicate: copmodel.SensorFootprintGeometry},
	} {
		if _, ok := latestPropertyValue(entity, field.predicate); ok {
			fields = append(fields, field.name)
		}
	}
	return fields
}

func sensorFootprintWarnings(entity graph.EntityState, mediaRef string, packetRef string) []string {
	warnings := make([]string, 0, 3)
	if strings.TrimSpace(mediaRef) == "" {
		warnings = append(warnings, "media reference missing")
	}
	if strings.TrimSpace(packetRef) == "" {
		warnings = append(warnings, "packet reference missing")
	}
	if len(latestPolygon(entity, copmodel.SensorFootprintGeometry)) == 0 {
		warnings = append(warnings, "footprint polygon not computed")
	}
	return warnings
}

func weatherObservationFromEntity(
	entity graph.EntityState,
	now time.Time,
	freshness time.Duration,
) (WeatherObservation, bool) {
	variable := latestStringProperty(entity, copmodel.WeatherVariable, "")
	value, valueOK := latestFloatProperty(entity, copmodel.WeatherValue)
	validTime, validOK := latestTimeProperty(entity, copmodel.WeatherValidTime)
	if variable == "" || !valueOK || !validOK {
		return WeatherObservation{}, false
	}

	updatedAt := latestObservedAt(entity, copmodel.ProvenanceObservedAt, copmodel.WeatherModelTime, copmodel.WeatherValidTime)
	modelTime, _ := latestTimeProperty(entity, copmodel.WeatherModelTime)
	freshUntil, _ := latestTimeProperty(entity, copmodel.WeatherFreshUntil)
	queryGeometry := latestStringProperty(entity, copmodel.WeatherQueryGeometry, "")
	position := optionalLatestPoint(entity, copmodel.WeatherQueryGeometry)
	source := stringProperty(entity, copmodel.ProvenanceSource, sourceFromEntityID(entity.ID))
	return WeatherObservation{
		ID:               entity.ID,
		Label:            weatherObservationLabel(entity, variable),
		Source:           source,
		Status:           weatherObservationStatus(now, updatedAt, freshUntil, freshness),
		Provider:         latestStringProperty(entity, copmodel.WeatherProvider, ""),
		QueryShape:       latestStringProperty(entity, copmodel.WeatherQueryShape, ""),
		QueryGeometryWKT: queryGeometry,
		Position:         position,
		ValidTime:        validTime,
		ModelTime:        modelTime,
		FreshUntil:       freshUntil,
		Variable:         variable,
		Value:            value,
		Unit:             latestStringProperty(entity, copmodel.WeatherUnit, ""),
		ClaimPosture:     "tactical weather graph readback; fixture/provider-shaped point evidence; no live-provider, cache, route-safety, or OGC conformance claim",
		Confidence:       confidence(entity),
		UpdatedAt:        updatedAt,
		Provenance: Provenance{
			Owner:     ownerForSource(source),
			SourceRef: latestStringProperty(entity, copmodel.ProvenanceSourceRef, ""),
			Observed:  updatedAt,
		},
	}, true
}

func weatherObservationLabel(entity graph.EntityState, variable string) string {
	provider := latestStringProperty(entity, copmodel.WeatherProvider, "")
	unit := latestStringProperty(entity, copmodel.WeatherUnit, "")
	value, ok := latestFloatProperty(entity, copmodel.WeatherValue)
	if provider == "" {
		provider = "weather"
	}
	if !ok {
		return provider + " " + variable
	}
	rendered := strconv.FormatFloat(value, 'f', -1, 64)
	if unit != "" {
		rendered += " " + unit
	}
	return provider + " " + variable + " " + rendered
}

func weatherObservationStatus(now time.Time, updatedAt time.Time, freshUntil time.Time, freshness time.Duration) string {
	if !freshUntil.IsZero() && !now.IsZero() && !freshUntil.Before(now.UTC()) {
		return "fresh"
	}
	if !freshUntil.IsZero() && !now.IsZero() && freshUntil.Before(now.UTC()) {
		return "stale"
	}
	return freshnessStatus("fresh", now, updatedAt, freshness)
}

func associationFromEntity(entity graph.EntityState, now time.Time, freshness time.Duration) (Association, bool) {
	primaryTrack := latestStringProperty(entity, copmodel.AssociationPrimaryTrack, "")
	candidateTrack := latestStringProperty(entity, copmodel.AssociationCandidateTrack, "")
	if primaryTrack == "" || candidateTrack == "" {
		return Association{}, false
	}
	updatedAt := latestObservedAt(entity, copmodel.AssociationObservedAt, copmodel.ProvenanceObservedAt)
	confidence := confidence(entity)
	if associationConfidence, ok := latestFloatProperty(entity, copmodel.AssociationConfidence); ok {
		confidence = associationConfidence
	}
	distanceMeters := optionalLatestFloat(entity, copmodel.AssociationDistanceMeters)
	timeDeltaSeconds := optionalLatestFloat(entity, copmodel.AssociationTimeDeltaSeconds)
	status := freshnessStatus(latestStringProperty(entity, copmodel.AssociationStatus, "unknown"), now, updatedAt, freshness)
	return Association{
		ID:               entity.ID,
		Label:            associationLabel(primaryTrack, candidateTrack, status),
		Kind:             latestStringProperty(entity, copmodel.AssociationKind, "track"),
		Source:           "fusion",
		Status:           status,
		PrimaryTrackID:   primaryTrack,
		CandidateTrackID: candidateTrack,
		Algorithm:        latestStringProperty(entity, copmodel.AssociationAlgorithm, ""),
		Reason:           latestStringProperty(entity, copmodel.AssociationReason, ""),
		DistanceMeters:   distanceMeters,
		TimeDeltaSeconds: timeDeltaSeconds,
		ClaimPosture:     "fusion association evidence; no source-track merge; no identity authority",
		Confidence:       confidence,
		UpdatedAt:        updatedAt,
		Provenance: Provenance{
			Owner:     ownerForSource("fusion"),
			SourceRef: latestStringProperty(entity, copmodel.ProvenanceSourceRef, ""),
			Observed:  updatedAt,
		},
	}, true
}

func associationReviewsByAssociationID(reviewEntities map[string]graph.EntityState) map[string]AssociationReview {
	reviews := make(map[string]AssociationReview, len(reviewEntities))
	for _, entity := range sortedEntities(reviewEntities) {
		review, ok := associationReviewFromEntity(entity)
		if !ok {
			continue
		}
		reviews[review.AssociationID] = review
	}
	return reviews
}

func associationReviewFromEntity(entity graph.EntityState) (AssociationReview, bool) {
	associationID := latestStringProperty(entity, copmodel.AssociationReviewAssociation, "")
	decision := latestStringProperty(entity, copmodel.AssociationReviewDecision, "")
	reviewedBy := latestStringProperty(entity, copmodel.AssociationReviewReviewedBy, "")
	reviewedAt := latestObservedAt(entity, copmodel.AssociationReviewReviewedAt, copmodel.ProvenanceObservedAt)
	if associationID == "" || decision == "" || reviewedBy == "" || reviewedAt.IsZero() {
		return AssociationReview{}, false
	}
	return AssociationReview{
		AssociationID:   associationID,
		Decision:        normalizeAssociationReviewDecision(decision),
		ReviewedBy:      reviewedBy,
		ReviewedAt:      reviewedAt,
		ReviewerRole:    latestStringProperty(entity, copmodel.AssociationReviewReviewerRole, DefaultAssociationReviewerRole),
		AuthorityScope:  latestStringProperty(entity, copmodel.AssociationReviewAuthorityScope, DefaultAssociationReviewAuthorityScope),
		AuthorityDomain: latestStringProperty(entity, copmodel.AssociationReviewAuthorityDomain, DefaultAssociationReviewAuthorityDomain),
		ConflictPolicy:  latestStringProperty(entity, copmodel.AssociationReviewConflictPolicy, DefaultAssociationReviewConflictPolicy),
		ConflictState:   latestStringProperty(entity, copmodel.AssociationReviewConflictState, DefaultAssociationReviewConflictState),
		Authenticated:   latestBoolProperty(entity, copmodel.AssociationReviewAuthenticated, false),
		Comment:         latestStringProperty(entity, copmodel.AssociationReviewComment, ""),
	}, true
}

func associationLabel(primaryTrack string, candidateTrack string, status string) string {
	prefix := "Candidate association"
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ambiguous":
		prefix = "Ambiguous association"
	case "stale":
		prefix = "Stale association evidence"
	case "unknown", "":
		prefix = "Association evidence"
	}
	return prefix + " " + instanceLabel(primaryTrack) + " -> " + instanceLabel(candidateTrack)
}

func hazardEvidenceDocument(entity graph.EntityState) (copmodel.HazardEvidenceDocument, bool) {
	value, ok := latestPropertyValue(entity, copmodel.HazardEvidence)
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

func hazardStatus(
	evidence copmodel.HazardEvidenceDocument,
	now time.Time,
	updatedAt time.Time,
	freshness time.Duration,
) string {
	capStatus := strings.ToLower(strings.TrimSpace(evidence.Status))
	if capStatus != "" && capStatus != "actual" {
		return "nonoperational." + capStatus
	}

	switch strings.ToLower(strings.TrimSpace(evidence.MessageType)) {
	case "cancel":
		return "cancelled"
	case "error":
		return "error"
	case "ack":
		return "acknowledged"
	}

	if expires, ok := evidenceTimestamp(evidence.Expires); ok && !now.IsZero() && !expires.After(now.UTC()) {
		return "expired"
	}

	status := "active"
	if strings.EqualFold(strings.TrimSpace(evidence.MessageType), "update") {
		status = "active.update"
	}
	return freshnessStatus(status, now, updatedAt, freshness)
}

func evidenceTimestamp(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
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

func latestPoint(entity graph.EntityState, predicate string) (GeoPoint, bool) {
	value, ok := latestPropertyValue(entity, predicate)
	if !ok {
		return GeoPoint{}, false
	}
	return geoPointFromWKT(value)
}

func optionalLatestPoint(entity graph.EntityState, predicate string) *GeoPoint {
	point, ok := latestPoint(entity, predicate)
	if !ok {
		return nil
	}
	return &point
}

func latestPolygon(entity graph.EntityState, predicate string) []GeoPoint {
	value, ok := latestPropertyValue(entity, predicate)
	if !ok {
		return nil
	}
	return geoPolygonFromWKT(value)
}

func optionalFloat(entity graph.EntityState, predicate string) *float64 {
	value, ok := latestPropertyValue(entity, predicate)
	if !ok {
		return nil
	}
	parsed, ok := floatFromAny(value)
	if !ok {
		return nil
	}
	return &parsed
}

func optionalLatestInt(entity graph.EntityState, predicate string) *int {
	value, ok := latestPropertyValue(entity, predicate)
	if !ok {
		return nil
	}
	parsed, ok := intFromAny(value)
	if !ok {
		return nil
	}
	return &parsed
}

func optionalLatestFloat(entity graph.EntityState, predicate string) *float64 {
	parsed, ok := latestFloatProperty(entity, predicate)
	if !ok {
		return nil
	}
	return &parsed
}

func optionalLatestTime(entity graph.EntityState, predicate string) *time.Time {
	parsed, ok := latestTimeProperty(entity, predicate)
	if !ok {
		return nil
	}
	return &parsed
}

func latestFloatProperty(entity graph.EntityState, predicate string) (float64, bool) {
	value, ok := latestPropertyValue(entity, predicate)
	if !ok {
		return 0, false
	}
	return floatFromAny(value)
}

func latestTimeProperty(entity graph.EntityState, predicate string) (time.Time, bool) {
	value, ok := latestPropertyValue(entity, predicate)
	if !ok {
		return time.Time{}, false
	}
	parsed, ok := timeFromAny(value)
	if !ok {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func latestBoolProperty(entity graph.EntityState, predicate string, fallback bool) bool {
	value, ok := latestPropertyValue(entity, predicate)
	if !ok {
		return fallback
	}
	parsed, ok := boolFromAny(value)
	if !ok {
		return fallback
	}
	return parsed
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
	if strings.Contains(entityID, ".cop.command.") {
		return "command"
	}
	if strings.Contains(entityID, ".cop.mavlink.") {
		return "mavlink"
	}
	if strings.Contains(entityID, ".cop.cap.") {
		return "cap"
	}
	if strings.Contains(entityID, ".cop.adsb.") {
		return "adsb"
	}
	if strings.Contains(entityID, ".cop.sapient.") {
		return "sapient"
	}
	if strings.Contains(entityID, ".cop.klv.") {
		return "klv"
	}
	if strings.Contains(entityID, ".cop.weather.") {
		return "weather"
	}
	if strings.Contains(entityID, ".cop.fusion.") {
		return "fusion"
	}
	return "graph"
}

func ownerForSource(source string) string {
	switch source {
	case "tak-cot":
		return copmodel.OwnerTAK
	case "mavlink":
		return copmodel.OwnerMAVLink
	case "command":
		return copmodel.OwnerCommand
	case "cap":
		return copmodel.OwnerCAP
	case "adsb":
		return copmodel.OwnerADSB
	case "sapient":
		return copmodel.OwnerSAPIENT
	case "klv":
		return copmodel.OwnerKLV
	case "weather":
		return copmodel.OwnerWeather
	case "fusion":
		return copmodel.OwnerFusion
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

func latestPropertyValue(entity graph.EntityState, predicate string) (any, bool) {
	var (
		value         any
		latest        time.Time
		selectedIndex int
		seen          bool
	)
	for index, triple := range entity.Triples {
		if triple.Predicate != predicate {
			continue
		}
		if !seen || triple.Timestamp.After(latest) || (triple.Timestamp.Equal(latest) && index > selectedIndex) {
			value = triple.Object
			latest = triple.Timestamp
			selectedIndex = index
			seen = true
		}
	}
	if seen {
		return value, true
	}
	return propertyValue(entity, predicate)
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

func latestStringProperty(entity graph.EntityState, predicate string, fallback string) string {
	value, ok := latestPropertyValue(entity, predicate)
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

func latestObservedAt(entity graph.EntityState, predicates ...string) time.Time {
	for _, predicate := range predicates {
		if value, ok := latestPropertyValue(entity, predicate); ok {
			if parsed, ok := timeFromAny(value); ok {
				return parsed.UTC()
			}
		}
	}
	return observedAt(entity, predicates...)
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

func geoPolygonFromWKT(value any) []GeoPoint {
	text, ok := stringFromAny(value)
	if !ok {
		return nil
	}
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "POLYGON((") || !strings.HasSuffix(text, "))") {
		return nil
	}
	body := strings.TrimSuffix(strings.TrimPrefix(text, "POLYGON(("), "))")
	rawPoints := strings.Split(body, ",")
	points := make([]GeoPoint, 0, len(rawPoints))
	for _, rawPoint := range rawPoints {
		parts := strings.Fields(strings.TrimSpace(rawPoint))
		if len(parts) != 2 {
			return nil
		}
		lon, lonErr := strconv.ParseFloat(parts[0], 64)
		lat, latErr := strconv.ParseFloat(parts[1], 64)
		if lonErr != nil || latErr != nil {
			return nil
		}
		points = append(points, GeoPoint{Lat: lat, Lon: lon})
	}
	if len(points) >= 4 && points[0] == points[len(points)-1] {
		points = points[:len(points)-1]
	}
	if len(points) < 3 {
		return nil
	}
	return points
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

func boolFromAny(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
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

func intFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), typed == float64(int(typed))
	case json.Number:
		parsed, err := typed.Int64()
		return int(parsed), err == nil
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
	if aircraftID := adsbIDFromNativeID(nativeID); aircraftID != "" {
		return aircraftID
	}
	if objectID := sapientObjectIDFromNativeID(nativeID); objectID != "" {
		return objectID
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

func sapientObjectIDFromNativeID(nativeID string) string {
	parts := strings.Split(strings.TrimSpace(nativeID), ".")
	if len(parts) < 3 || parts[0] != "sapient" || parts[1] != "object" {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(parts[2]))
}

func adsbIDFromNativeID(nativeID string) string {
	parts := strings.Split(strings.TrimSpace(nativeID), ".")
	if len(parts) < 3 || parts[0] != "adsb" || parts[1] != "icao24" {
		return ""
	}
	for i := 2; i < len(parts)-1; i++ {
		if parts[i] == "callsign" && strings.TrimSpace(parts[i+1]) != "" {
			return strings.ToUpper(strings.TrimSpace(parts[i+1]))
		}
	}
	return strings.ToUpper(strings.TrimSpace(parts[2]))
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

func normalizeGraphDiscovery(discovery GraphDiscoveryConfig) GraphDiscoveryConfig {
	if discovery.Limit <= 0 {
		discovery.Limit = DefaultGraphDiscoveryLimit
	}
	if len(discovery.Scopes) == 0 {
		discovery.Scopes = []GraphDiscoveryScope{{Org: "c360", Platform: "edge"}}
	}
	out := make([]GraphDiscoveryScope, 0, len(discovery.Scopes))
	seen := make(map[string]struct{}, len(discovery.Scopes))
	for _, scope := range discovery.Scopes {
		if scope.Org == "" {
			scope.Org = "c360"
		}
		if scope.Platform == "" {
			scope.Platform = "edge"
		}
		key := scope.Org + "\x00" + scope.Platform
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, scope)
	}
	discovery.Scopes = out
	return discovery
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

func graphEntityPrefix(org, platform, system, entityType string) string {
	return strings.Join([]string{org, platform, "cop", system, entityType}, ".")
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
