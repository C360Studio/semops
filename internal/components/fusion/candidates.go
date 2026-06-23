package fusion

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	fusionassociation "github.com/c360studio/semops/internal/fusion/association"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

const (
	SubjectGraphQueryPrefix = "graph.query.prefix"

	DefaultCandidateProducerInterval           = 5 * time.Second
	DefaultCandidateProducerQueryTimeout       = 2 * time.Second
	DefaultCandidateProducerLimitPerSource     = 64
	DefaultCandidateProducerMaxPairComparisons = 256
	DefaultCandidateProducerMaxBatches         = 32
)

var DefaultCandidateProducerSources = []CandidateSourceScope{
	{Org: "c360", Platform: "edge", Source: "mavlink"},
	{Org: "c360", Platform: "edge", Source: "tak"},
	{Org: "c360", Platform: "edge", Source: "adsb"},
	{Org: "c360", Platform: "edge", Source: "sapient"},
}

type PrefixRequester interface {
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
}

type classifiedPrefixRequester interface {
	RequestClassified(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
}

type CandidateSourceScope struct {
	Org      string
	Platform string
	Source   string
}

type CandidateProducerConfig struct {
	Name               string
	CandidateSubject   string
	QuerySubject       string
	Registry           *payloadregistry.Registry
	Requester          PrefixRequester
	Sources            []CandidateSourceScope
	PollInterval       time.Duration
	QueryTimeout       time.Duration
	LimitPerSource     int
	MaxPairComparisons int
	MaxBatches         int
	Clock              func() time.Time
}

type CandidateProducerComponent struct {
	cfg CandidateProducerConfig
	bus Bus

	mu      sync.Mutex
	state   component.State
	cancel  context.CancelFunc
	metrics flowCounters
}

func NewCandidateProducerComponent(
	cfg CandidateProducerConfig,
	bus Bus,
) (*CandidateProducerComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-processor-fusion-candidates"
	}
	if cfg.CandidateSubject == "" {
		cfg.CandidateSubject = DefaultCandidateSubject
	}
	if cfg.QuerySubject == "" {
		cfg.QuerySubject = SubjectGraphQueryPrefix
	}
	if cfg.Registry == nil {
		cfg.Registry = payloadregistry.New()
	}
	if cfg.Requester == nil {
		return nil, fmt.Errorf("fusion candidate producer requires a graph prefix requester")
	}
	if bus == nil {
		return nil, fmt.Errorf("fusion candidate producer requires a bus")
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = DefaultCandidateProducerInterval
	}
	if cfg.QueryTimeout <= 0 {
		cfg.QueryTimeout = DefaultCandidateProducerQueryTimeout
	}
	if cfg.LimitPerSource <= 0 {
		cfg.LimitPerSource = DefaultCandidateProducerLimitPerSource
	}
	if cfg.MaxPairComparisons <= 0 {
		cfg.MaxPairComparisons = DefaultCandidateProducerMaxPairComparisons
	}
	if cfg.MaxBatches <= 0 {
		cfg.MaxBatches = DefaultCandidateProducerMaxBatches
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	cfg.Sources = normalizeCandidateSourceScopes(cfg.Sources)
	return &CandidateProducerComponent{cfg: cfg, bus: bus, state: component.StateCreated}, nil
}

func (c *CandidateProducerComponent) Initialize() error {
	if err := RegisterPayloads(c.cfg.Registry); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == component.StateCreated {
		c.state = component.StateInitialized
	}
	return nil
}

func (c *CandidateProducerComponent) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.Initialize(); err != nil {
		return err
	}
	c.mu.Lock()
	if c.state == component.StateStarted {
		c.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.state = component.StateStarted
	c.metrics.startedAt = c.cfg.Clock().UTC()
	c.mu.Unlock()

	go c.run(runCtx)
	return nil
}

func (c *CandidateProducerComponent) Stop(time.Duration) error {
	c.mu.Lock()
	cancel := c.cancel
	c.cancel = nil
	if c.state == component.StateStarted {
		c.state = component.StateStopped
	}
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (c *CandidateProducerComponent) ScanAndPublish(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.Initialize(); err != nil {
		return err
	}
	now := c.cfg.Clock().UTC()
	observationsBySource, err := c.discoverObservations(ctx)
	if err != nil {
		c.recordError(err)
		return err
	}
	batches := c.candidateBatches(observationsBySource, now)
	for _, payload := range batches {
		if err := payload.Validate(); err != nil {
			c.recordError(err)
			return err
		}
		data, err := marshalBaseMessage(CandidateBatchType, payload, c.cfg.Name, now)
		if err != nil {
			c.recordError(err)
			return err
		}
		if err := c.bus.Publish(ctx, c.cfg.CandidateSubject, data); err != nil {
			wrapped := fmt.Errorf("publish fusion candidate batch: %w", err)
			c.recordError(wrapped)
			return wrapped
		}
		c.recordMessage(len(data), now)
	}
	return nil
}

func (c *CandidateProducerComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "Fusion graph-discovery track-candidate producer",
		Version:     "v0.1.0",
	}
}

func (c *CandidateProducerComponent) InputPorts() []component.Port {
	return []component.Port{
		{
			Name:        "discovery_timer",
			Direction:   component.DirectionInput,
			Required:    true,
			Description: "Periodic graph-discovery trigger",
			Config: component.TimerPort{
				Interval: c.pollInterval().String(),
				Interface: &component.InterfaceContract{
					Type:    "component.Timer",
					Version: "v1",
				},
			},
		},
	}
}

func (c *CandidateProducerComponent) OutputPorts() []component.Port {
	return []component.Port{
		{
			Name:        "graph_prefix_query",
			Direction:   component.DirectionOutput,
			Required:    true,
			Description: "SemStreams graph prefix query for source-owned tracks",
			Config: component.NATSRequestPort{
				Subject: c.cfg.QuerySubject,
				Timeout: c.queryTimeout().String(),
				Interface: &component.InterfaceContract{
					Type:    "graph.PrefixQueryRequest",
					Version: "v1",
				},
			},
		},
		streamPort("track_candidates", component.DirectionOutput, c.cfg.CandidateSubject, CandidateBatchType),
	}
}

func (c *CandidateProducerComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"candidate_subject": stringProperty("SemStreams subject carrying fusion track-candidate batches", c.cfg.CandidateSubject),
			"query_subject":     stringProperty("SemStreams graph prefix query subject", c.cfg.QuerySubject),
			"poll_interval":     stringProperty("Graph discovery polling interval", c.pollInterval().String()),
			"query_timeout":     stringProperty("Graph prefix query timeout", c.queryTimeout().String()),
			"limit_per_source":  intProperty("Maximum source-owned tracks discovered per source prefix", c.limitPerSource()),
			"max_pair_comparisons": intProperty(
				"Maximum primary-candidate comparisons per published batch",
				c.maxPairComparisons(),
			),
			"max_batches": intProperty("Maximum candidate batches emitted by one discovery scan", c.maxBatches()),
			"sources": component.PropertySchema{
				Type:        "array",
				Description: "Graph source prefixes considered for candidate production",
				Default:     candidateSourceScopeNames(c.cfg.Sources),
				Items:       &component.PropertySchema{Type: "string"},
			},
		},
		Required: []string{"candidate_subject", "query_subject", "sources"},
	}
}

func (c *CandidateProducerComponent) Health() component.HealthStatus {
	return c.metrics.health(c.state)
}

func (c *CandidateProducerComponent) DataFlow() component.FlowMetrics {
	return c.metrics.flow()
}

func (c *CandidateProducerComponent) run(ctx context.Context) {
	ticker := time.NewTicker(c.pollInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = c.ScanAndPublish(ctx)
		}
	}
}

func (c *CandidateProducerComponent) discoverObservations(
	ctx context.Context,
) (map[string][]fusionassociation.TrackObservation, error) {
	result := make(map[string][]fusionassociation.TrackObservation)
	for _, scope := range c.cfg.Sources {
		entities, _, err := c.queryPrefix(ctx, candidateTrackPrefix(scope))
		if err != nil {
			return nil, err
		}
		for _, entity := range entities {
			obs, ok := trackObservationFromEntity(entity, scope, c.cfg.Clock)
			if ok {
				result[scope.Source] = append(result[scope.Source], obs)
			}
		}
	}
	for source := range result {
		sortTrackObservations(result[source])
	}
	return result, nil
}

func (c *CandidateProducerComponent) queryPrefix(
	ctx context.Context,
	prefix string,
) ([]graph.EntityState, bool, error) {
	limit := c.limitPerSource()
	entities := make([]graph.EntityState, 0, minInt(limit, 64))
	remaining := limit
	cursor := ""
	for {
		pageLimit := prefixPageLimit(remaining)
		if pageLimit <= 0 {
			return entities, cursor != "", nil
		}
		page, err := c.queryPrefixPage(ctx, prefix, cursor, pageLimit)
		if err != nil {
			return nil, false, err
		}
		if len(page.Entities) > remaining {
			entities = append(entities, page.Entities[:remaining]...)
			return entities, true, nil
		}
		entities = append(entities, page.Entities...)
		remaining -= len(page.Entities)
		if page.NextCursor == "" {
			return entities, false, nil
		}
		if remaining <= 0 {
			return entities, true, nil
		}
		if len(page.Entities) == 0 {
			return nil, false, fmt.Errorf("query graph prefix %s: empty page returned continuation cursor", prefix)
		}
		if page.NextCursor == cursor {
			return nil, false, fmt.Errorf("query graph prefix %s: repeated continuation cursor %q", prefix, cursor)
		}
		cursor = page.NextCursor
	}
}

func (c *CandidateProducerComponent) queryPrefixPage(
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
	var response []byte
	if classified, ok := c.cfg.Requester.(classifiedPrefixRequester); ok {
		response, err = classified.RequestClassified(ctx, c.cfg.QuerySubject, body, c.queryTimeout())
	} else {
		response, err = c.cfg.Requester.Request(ctx, c.cfg.QuerySubject, body, c.queryTimeout())
	}
	if err != nil {
		return graph.PrefixQueryResponse{}, fmt.Errorf("query graph prefix %s: %w", prefix, err)
	}
	if len(response) == 0 || isNotFoundText(string(response)) {
		return graph.PrefixQueryResponse{}, nil
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

func (c *CandidateProducerComponent) candidateBatches(
	observations map[string][]fusionassociation.TrackObservation,
	now time.Time,
) []*CandidateBatchPayload {
	sources := candidateSourceScopeNames(c.cfg.Sources)
	batches := make([]*CandidateBatchPayload, 0)
	for i := 0; i < len(sources); i++ {
		primary := observations[sources[i]]
		if len(primary) == 0 {
			continue
		}
		for j := i + 1; j < len(sources); j++ {
			candidates := observations[sources[j]]
			if len(candidates) == 0 {
				continue
			}
			batches = append(batches, c.sourcePairBatches(primary, candidates, sources[i], sources[j], now)...)
			if len(batches) >= c.maxBatches() {
				return batches[:c.maxBatches()]
			}
		}
	}
	return batches
}

func (c *CandidateProducerComponent) sourcePairBatches(
	primary []fusionassociation.TrackObservation,
	candidates []fusionassociation.TrackObservation,
	primarySource string,
	candidateSource string,
	now time.Time,
) []*CandidateBatchPayload {
	maxComparisons := c.maxPairComparisons()
	if maxComparisons <= 0 {
		maxComparisons = 1
	}
	batches := make([]*CandidateBatchPayload, 0)
	for pStart := 0; pStart < len(primary); {
		primaryChunkSize := maxInt(1, maxComparisons/maxInt(1, minInt(len(candidates), maxComparisons)))
		pEnd := minInt(len(primary), pStart+primaryChunkSize)
		for cStart := 0; cStart < len(candidates); {
			candidateChunkSize := maxInt(1, maxComparisons/maxInt(1, pEnd-pStart))
			cEnd := minInt(len(candidates), cStart+candidateChunkSize)
			batchID := candidateBatchID(primarySource, candidateSource, len(batches), now)
			batches = append(batches, NewCandidateBatchPayload(
				c.cfg.Name,
				batchID,
				now,
				primary[pStart:pEnd],
				candidates[cStart:cEnd],
			))
			cStart = cEnd
			if len(batches) >= c.maxBatches() {
				return batches
			}
		}
		pStart = pEnd
	}
	return batches
}

func (c *CandidateProducerComponent) recordMessage(size int, now time.Time) {
	c.metrics.recordMessage(size, now)
}

func (c *CandidateProducerComponent) recordError(err error) {
	c.metrics.recordError(err)
}

func (c *CandidateProducerComponent) pollInterval() time.Duration {
	if c.cfg.PollInterval > 0 {
		return c.cfg.PollInterval
	}
	return DefaultCandidateProducerInterval
}

func (c *CandidateProducerComponent) queryTimeout() time.Duration {
	if c.cfg.QueryTimeout > 0 {
		return c.cfg.QueryTimeout
	}
	return DefaultCandidateProducerQueryTimeout
}

func (c *CandidateProducerComponent) limitPerSource() int {
	if c.cfg.LimitPerSource > 0 {
		return c.cfg.LimitPerSource
	}
	return DefaultCandidateProducerLimitPerSource
}

func (c *CandidateProducerComponent) maxPairComparisons() int {
	if c.cfg.MaxPairComparisons > 0 {
		return c.cfg.MaxPairComparisons
	}
	return DefaultCandidateProducerMaxPairComparisons
}

func (c *CandidateProducerComponent) maxBatches() int {
	if c.cfg.MaxBatches > 0 {
		return c.cfg.MaxBatches
	}
	return DefaultCandidateProducerMaxBatches
}

func normalizeCandidateSourceScopes(scopes []CandidateSourceScope) []CandidateSourceScope {
	if len(scopes) == 0 {
		scopes = DefaultCandidateProducerSources
	}
	out := make([]CandidateSourceScope, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope.Org = nonEmptyString(strings.TrimSpace(scope.Org), "c360")
		scope.Platform = nonEmptyString(strings.TrimSpace(scope.Platform), "edge")
		scope.Source = strings.ToLower(strings.TrimSpace(scope.Source))
		if scope.Source == "" {
			continue
		}
		key := strings.Join([]string{scope.Org, scope.Platform, scope.Source}, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, scope)
	}
	if len(out) == 0 {
		return normalizeCandidateSourceScopes(DefaultCandidateProducerSources)
	}
	return out
}

func candidateSourceScopeNames(scopes []CandidateSourceScope) []string {
	scopes = normalizeCandidateSourceScopes(scopes)
	names := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		names = append(names, scope.Source)
	}
	return names
}

func candidateTrackPrefix(scope CandidateSourceScope) string {
	return strings.Join([]string{
		scope.Org,
		scope.Platform,
		"cop",
		scope.Source,
		cop.EntityTrack,
	}, ".")
}

func trackObservationFromEntity(
	entity graph.EntityState,
	scope CandidateSourceScope,
	clock func() time.Time,
) (fusionassociation.TrackObservation, bool) {
	parsed, err := message.ParseEntityID(entity.ID)
	if err != nil || parsed.Domain != "cop" || parsed.Type != cop.EntityTrack {
		return fusionassociation.TrackObservation{}, false
	}
	positionValue, ok := latestPropertyValue(entity, cop.TrackPosition)
	if !ok {
		return fusionassociation.TrackObservation{}, false
	}
	position, ok := geoPointFromWKT(positionValue)
	if !ok {
		return fusionassociation.TrackObservation{}, false
	}
	observedAt := observedAt(entity, clock, cop.TrackObservedAt, cop.ProvenanceObservedAt)
	if observedAt.IsZero() {
		return fusionassociation.TrackObservation{}, false
	}
	source := latestStringProperty(entity, cop.ProvenanceSource, "")
	if source == "" {
		source = nonEmptyString(scope.Source, parsed.System)
	}
	return fusionassociation.TrackObservation{
		ID:         entity.ID,
		Source:     source,
		NativeID:   latestStringProperty(entity, cop.TrackNativeID, parsed.Instance),
		Position:   position,
		ObservedAt: observedAt,
		Confidence: confidence(entity),
		SourceRef:  latestStringProperty(entity, cop.ProvenanceSourceRef, ""),
	}, true
}

func latestPropertyValue(entity graph.EntityState, predicate string) (any, bool) {
	var latest message.Triple
	var found bool
	for _, triple := range entity.Triples {
		if triple.Predicate != predicate {
			continue
		}
		if !found || triple.Timestamp.After(latest.Timestamp) {
			latest = triple
			found = true
		}
	}
	if !found {
		return nil, false
	}
	return latest.Object, true
}

func latestStringProperty(entity graph.EntityState, predicate string, fallback string) string {
	value, ok := latestPropertyValue(entity, predicate)
	if !ok {
		return fallback
	}
	text, ok := stringFromAny(value)
	if !ok || strings.TrimSpace(text) == "" {
		return fallback
	}
	return strings.TrimSpace(text)
}

func observedAt(entity graph.EntityState, clock func() time.Time, predicates ...string) time.Time {
	for _, predicate := range predicates {
		if value, ok := latestPropertyValue(entity, predicate); ok {
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
	if latest.IsZero() && clock != nil {
		latest = clock()
	}
	return latest.UTC()
}

func geoPointFromWKT(value any) (fusionassociation.GeoPoint, bool) {
	text, ok := stringFromAny(value)
	if !ok {
		return fusionassociation.GeoPoint{}, false
	}
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "POINT(") || !strings.HasSuffix(text, ")") {
		return fusionassociation.GeoPoint{}, false
	}
	parts := strings.Fields(strings.TrimSuffix(strings.TrimPrefix(text, "POINT("), ")"))
	if len(parts) != 2 {
		return fusionassociation.GeoPoint{}, false
	}
	lon, lonErr := strconv.ParseFloat(parts[0], 64)
	lat, latErr := strconv.ParseFloat(parts[1], 64)
	if lonErr != nil || latErr != nil || math.IsNaN(lat) || math.IsNaN(lon) {
		return fusionassociation.GeoPoint{}, false
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return fusionassociation.GeoPoint{}, false
	}
	return fusionassociation.GeoPoint{Lat: lat, Lon: lon}, true
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

func confidence(entity graph.EntityState) float64 {
	if value, ok := latestPropertyValue(entity, cop.ProvenanceConfidence); ok {
		if parsed, ok := floatFromAny(value); ok {
			return clamp01(parsed)
		}
	}
	var latest message.Triple
	var found bool
	for _, triple := range entity.Triples {
		if !found || triple.Timestamp.After(latest.Timestamp) {
			latest = triple
			found = true
		}
	}
	if found && latest.Confidence > 0 {
		return clamp01(latest.Confidence)
	}
	return 0.5
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func sortTrackObservations(observations []fusionassociation.TrackObservation) {
	sort.Slice(observations, func(i, j int) bool {
		if observations[i].ObservedAt.Equal(observations[j].ObservedAt) {
			return observations[i].ID < observations[j].ID
		}
		return observations[i].ObservedAt.After(observations[j].ObservedAt)
	})
}

func candidateBatchID(primarySource string, candidateSource string, index int, now time.Time) string {
	return fmt.Sprintf(
		"fusion-candidates.%s-to-%s.%s.%03d",
		entityToken(primarySource),
		entityToken(candidateSource),
		now.UTC().Format("20060102T150405.000000000Z"),
		index,
	)
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

func isNotFoundText(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	return normalized == "not found" || strings.Contains(normalized, "not_found")
}

func intProperty(description string, fallback int) component.PropertySchema {
	return component.PropertySchema{Type: "int", Description: description, Default: fallback}
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func nonEmptyString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func entityToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		":", "-",
		"/", "-",
		"\\", "-",
		" ", "-",
		"_", "-",
	)
	value = replacer.Replace(value)
	value = strings.Trim(value, ".-")
	if value == "" {
		return "unknown"
	}
	return value
}
