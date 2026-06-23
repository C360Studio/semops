package weather

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	weathercodec "github.com/c360studio/semops/pkg/adapters/weather"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

type MutationKind string

const (
	MutationCreate MutationKind = "create"
	MutationUpdate MutationKind = "update"
)

type Config struct {
	Org         string
	Platform    string
	OwnerTokens map[string]ownership.OwnerToken
	TraceID     string
	Confidence  float64
}

type Observation struct {
	NativeID         string
	Provider         string
	QueryShape       string
	QueryGeometryWKT string
	ValidTime        time.Time
	ModelTime        time.Time
	FreshUntil       *time.Time
	Variable         string
	Value            float64
	Unit             string
	SourceRef        string
}

type Projector struct {
	cfg              Config
	bornObservations map[string]struct{}
}

type Plan struct {
	Mutations []Mutation
}

type Mutation struct {
	Kind   MutationKind
	Create graph.CreateEntityWithTriplesRequest
	Update graph.UpdateEntityWithTriplesRequest
}

func NewProjector(cfg Config) *Projector {
	if cfg.Org == "" {
		cfg.Org = "c360"
	}
	if cfg.Platform == "" {
		cfg.Platform = "edge"
	}
	if cfg.Confidence == 0 {
		cfg.Confidence = 0.75
	}
	cfg.OwnerTokens = cloneOwnerTokens(cfg.OwnerTokens)
	return &Projector{
		cfg:              cfg,
		bornObservations: make(map[string]struct{}),
	}
}

func (p *Projector) ProjectObservation(observation Observation) (Plan, error) {
	return p.projectObservation(observation, cloneStringSet(p.bornObservations))
}

func (p *Projector) ProjectObservations(observations []Observation) (Plan, error) {
	bornObservations := cloneStringSet(p.bornObservations)
	var plan Plan
	for _, observation := range observations {
		next, err := p.projectObservation(observation, bornObservations)
		if err != nil {
			return Plan{}, err
		}
		plan.Mutations = append(plan.Mutations, next.Mutations...)
	}
	return plan, nil
}

func (p *Projector) projectObservation(observation Observation, bornObservations map[string]struct{}) (Plan, error) {
	if p == nil {
		return Plan{}, fmt.Errorf("weather projector is nil")
	}
	if err := observation.validate(); err != nil {
		return Plan{}, err
	}

	observationID := p.observationID(observation.NativeID)
	triples := p.observationTriples(observationID, observation)
	if len(triples) == 0 {
		return Plan{}, nil
	}

	if _, ok := bornObservations[observationID]; !ok {
		bornObservations[observationID] = struct{}{}
		return Plan{Mutations: []Mutation{{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          observationID,
					MessageType: messageType(cop.WeatherObservationContract().MessageType),
					UpdatedAt:   entityUpdatedAt(observation),
				},
				Triples:         triples,
				IndexingProfile: cop.WeatherObservationContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerWeather),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-weather-observation", observation.NativeID),
			},
		}}}, nil
	}

	return Plan{Mutations: []Mutation{{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: observationID,
			},
			AddTriples:      triples,
			IndexingProfile: cop.WeatherObservationContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerWeather),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("update-weather-observation", observation.NativeID),
		},
	}}}, nil
}

func (p *Projector) MarkBornForPlan(plan Plan) int {
	if p == nil {
		return 0
	}
	var marked int
	for _, mutation := range plan.Mutations {
		if mutation.Kind != MutationCreate || mutation.Create.Entity == nil || mutation.Create.Entity.ID == "" {
			continue
		}
		if _, ok := p.bornObservations[mutation.Create.Entity.ID]; ok {
			continue
		}
		p.bornObservations[mutation.Create.Entity.ID] = struct{}{}
		marked++
	}
	return marked
}

func (p *Projector) MarkBornForObservation(observation Observation, entityID string) bool {
	if p == nil || entityID == "" || entityID != p.observationID(observation.NativeID) {
		return false
	}
	p.bornObservations[entityID] = struct{}{}
	return true
}

func (p *Projector) observationTriples(observationID string, observation Observation) []message.Triple {
	when := provenanceObservedAt(observation)
	triples := []message.Triple{
		p.triple(observationID, cop.WeatherNativeID, observation.NativeID, when),
		p.triple(observationID, cop.WeatherProvider, observation.Provider, when),
		p.triple(observationID, cop.WeatherQueryShape, observation.QueryShape, when),
		p.triple(observationID, cop.WeatherQueryGeometry, observation.QueryGeometryWKT, when),
		p.triple(observationID, cop.WeatherValidTime, observation.ValidTime.UTC(), when),
		p.triple(observationID, cop.WeatherVariable, observation.Variable, when),
		p.triple(observationID, cop.WeatherValue, observation.Value, when),
		p.triple(observationID, cop.ProvenanceSource, "weather", when),
		p.triple(observationID, cop.ProvenanceConfidence, p.cfg.Confidence, when),
		p.triple(observationID, cop.ProvenanceObservedAt, when, when),
	}
	if !observation.ModelTime.IsZero() {
		triples = append(triples, p.triple(observationID, cop.WeatherModelTime, observation.ModelTime.UTC(), when))
	}
	if observation.FreshUntil != nil {
		triples = append(triples, p.triple(observationID, cop.WeatherFreshUntil, observation.FreshUntil.UTC(), when))
	}
	if observation.Unit != "" {
		triples = append(triples, p.triple(observationID, cop.WeatherUnit, observation.Unit, when))
	}
	if observation.SourceRef != "" {
		triples = append(triples, p.triple(observationID, cop.ProvenanceSourceRef, observation.SourceRef, when))
	}
	return triples
}

func (p *Projector) triple(subject string, predicate string, object any, when time.Time) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "weather",
		Timestamp:  when.UTC(),
		Confidence: p.cfg.Confidence,
	}
}

func (p *Projector) observationID(nativeID string) string {
	return EntityID(p.cfg.Org, p.cfg.Platform, nativeID)
}

func EntityID(org, platform, nativeID string) string {
	return strings.Join([]string{
		entityToken(org),
		entityToken(platform),
		"cop",
		"weather",
		cop.EntityWeatherObservation,
		entityToken(nativeID),
	}, ".")
}

func ObservationsFromPointForecast(
	forecast weathercodec.PointForecast,
	sourceRef string,
	modelTime time.Time,
	freshness time.Duration,
) ([]Observation, error) {
	geometry := wktPoint(forecast.Latitude, forecast.Longitude)
	return observationsFromSamples(
		forecast.Provider,
		forecast.QueryShape,
		geometry,
		sourceRef,
		modelTime,
		freshness,
		forecast.Units,
		forecast.Samples,
	)
}

func ObservationsFromSpatialForecast(
	forecast weathercodec.SpatialForecast,
	sourceRef string,
	modelTime time.Time,
	freshness time.Duration,
) ([]Observation, error) {
	return observationsFromSamples(
		forecast.Provider,
		forecast.QueryShape,
		forecast.QueryGeometryWKT,
		sourceRef,
		modelTime,
		freshness,
		forecast.Units,
		forecast.Samples,
	)
}

func observationsFromSamples(
	provider string,
	queryShape string,
	geometry string,
	sourceRef string,
	modelTime time.Time,
	freshness time.Duration,
	units map[string]string,
	samples []weathercodec.WeatherSample,
) ([]Observation, error) {
	if strings.TrimSpace(provider) == "" {
		return nil, fmt.Errorf("weather forecast provider is required")
	}
	if strings.TrimSpace(queryShape) == "" {
		return nil, fmt.Errorf("weather forecast query shape is required")
	}
	if strings.TrimSpace(geometry) == "" {
		return nil, fmt.Errorf("weather forecast query geometry is required")
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("weather forecast samples are required")
	}

	var observations []Observation
	for _, sample := range samples {
		if sample.Time.IsZero() {
			return nil, fmt.Errorf("weather sample time is required")
		}
		for _, variable := range sampleVariables(sample) {
			observation := Observation{
				NativeID:         nativeID(provider, queryShape, geometry, sample.Time, variable.name),
				Provider:         strings.TrimSpace(provider),
				QueryShape:       strings.TrimSpace(queryShape),
				QueryGeometryWKT: strings.TrimSpace(geometry),
				ValidTime:        sample.Time.UTC(),
				ModelTime:        modelTime.UTC(),
				FreshUntil:       freshUntil(modelTime, sample.Time, freshness),
				Variable:         variable.name,
				Value:            variable.value,
				Unit:             strings.TrimSpace(units[variable.name]),
				SourceRef:        sourceRef,
			}
			observations = append(observations, observation)
		}
	}
	return observations, nil
}

func sampleVariables(sample weathercodec.WeatherSample) []sampleVariable {
	var variables []sampleVariable
	appendFloat := func(name string, value *float64) {
		if value != nil {
			variables = append(variables, sampleVariable{name: name, value: *value})
		}
	}
	appendFloat("temperature_2m", sample.TemperatureC)
	appendFloat("precipitation", sample.PrecipitationMM)
	appendFloat("visibility", sample.VisibilityM)
	appendFloat("surface_pressure", sample.SurfacePressureHPA)
	appendFloat("wind_speed_10m", sample.WindSpeed10MKPH)
	appendFloat("wind_gusts_10m", sample.WindGusts10MKPH)
	appendFloat("wind_direction_10m", sample.WindDirection10Deg)
	if sample.WeatherCode != nil {
		variables = append(variables, sampleVariable{name: "weather_code", value: float64(*sample.WeatherCode)})
	}
	return variables
}

type sampleVariable struct {
	name  string
	value float64
}

func (o Observation) validate() error {
	if strings.TrimSpace(o.NativeID) == "" {
		return fmt.Errorf("weather observation native_id is required")
	}
	if strings.TrimSpace(o.Provider) == "" {
		return fmt.Errorf("weather observation provider is required")
	}
	if strings.TrimSpace(o.QueryShape) == "" {
		return fmt.Errorf("weather observation query_shape is required")
	}
	if strings.TrimSpace(o.QueryGeometryWKT) == "" {
		return fmt.Errorf("weather observation query_geometry is required")
	}
	if o.ValidTime.IsZero() {
		return fmt.Errorf("weather observation valid_time is required")
	}
	if strings.TrimSpace(o.Variable) == "" {
		return fmt.Errorf("weather observation variable is required")
	}
	if math.IsNaN(o.Value) || math.IsInf(o.Value, 0) {
		return fmt.Errorf("weather observation value must be finite")
	}
	return nil
}

func freshUntil(modelTime time.Time, validTime time.Time, freshness time.Duration) *time.Time {
	if freshness <= 0 {
		return nil
	}
	base := modelTime
	if base.IsZero() {
		base = validTime
	}
	fresh := base.UTC().Add(freshness)
	return &fresh
}

func provenanceObservedAt(observation Observation) time.Time {
	if !observation.ModelTime.IsZero() {
		return observation.ModelTime.UTC()
	}
	if !observation.ValidTime.IsZero() {
		return observation.ValidTime.UTC()
	}
	return time.Time{}
}

func entityUpdatedAt(observation Observation) time.Time {
	if !observation.ValidTime.IsZero() {
		return observation.ValidTime.UTC()
	}
	return provenanceObservedAt(observation)
}

func nativeID(provider string, queryShape string, geometry string, validTime time.Time, variable string) string {
	return strings.Join([]string{
		"weather",
		entityToken(provider),
		entityToken(queryShape),
		shortHash(geometry),
		validTime.UTC().Format("20060102t150405z"),
		entityToken(variable),
	}, ".")
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])[:12]
}

func wktPoint(lat, lon float64) string {
	return "POINT(" + coord(lon) + " " + coord(lat) + ")"
}

func coord(v float64) string {
	return strconv.FormatFloat(v, 'f', 7, 64)
}

func (p *Projector) ownerToken(owner string) string {
	if p == nil || p.cfg.OwnerTokens == nil {
		return ""
	}
	return p.cfg.OwnerTokens[owner].Wire()
}

func entityToken(value string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.TrimSpace(value) {
		r = unicode.ToLower(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "unknown"
	}
	return token
}

func requestID(prefix, nativeID string) string {
	return prefix + "-" + entityToken(nativeID)
}

func messageType(key string) message.Type {
	parts := strings.Split(key, ".")
	if len(parts) < 3 {
		return message.Type{Domain: key, Category: "unknown", Version: "v1"}
	}
	return message.Type{
		Domain:   parts[0],
		Category: strings.Join(parts[1:len(parts)-1], "."),
		Version:  parts[len(parts)-1],
	}
}

func cloneStringSet(values map[string]struct{}) map[string]struct{} {
	if values == nil {
		return map[string]struct{}{}
	}
	out := make(map[string]struct{}, len(values))
	for key := range values {
		out[key] = struct{}{}
	}
	return out
}

func cloneOwnerTokens(tokens map[string]ownership.OwnerToken) map[string]ownership.OwnerToken {
	if tokens == nil {
		return nil
	}
	out := make(map[string]ownership.OwnerToken, len(tokens))
	for owner, token := range tokens {
		out[owner] = token
	}
	return out
}
