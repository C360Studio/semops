// Package association scores source-owned tracks for derived fusion evidence.
package association

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/c360studio/semops/pkg/cop"
)

const (
	DefaultAlgorithm         = "semops.association.geotemporal.v1"
	DefaultMaxDistanceMeters = 250
	DefaultMaxTimeDelta      = 10 * time.Second
	DefaultMinConfidence     = 0.65
	DefaultAmbiguityMargin   = 0.05
)

type GeoPoint struct {
	Lat float64
	Lon float64
}

type TrackObservation struct {
	ID         string
	Source     string
	NativeID   string
	Position   GeoPoint
	ObservedAt time.Time
	Confidence float64
	SourceRef  string
}

type Config struct {
	Org               string
	Platform          string
	Algorithm         string
	MaxDistanceMeters float64
	MaxTimeDelta      time.Duration
	MinConfidence     float64
	AmbiguityMargin   float64
}

type Evidence struct {
	EntityID             string
	PrimaryTrackID       string
	CandidateTrackID     string
	Status               string
	Confidence           float64
	Algorithm            string
	DistanceMeters       float64
	TimeDelta            time.Duration
	ObservedAt           time.Time
	Reasons              []string
	PrimarySourceRef     string
	CandidateSourceRef   string
	PrimaryTrackNative   string
	CandidateTrackNative string
}

// Associate returns the strongest cross-source association evidence for each
// primary track. Ambiguous candidates stay explicit rather than merging source
// track state.
func Associate(primary, candidates []TrackObservation, cfg Config) []Evidence {
	cfg = normalizeConfig(cfg)
	scoredByPrimary := make(map[string][]Evidence, len(primary))
	for _, left := range primary {
		if !validTrack(left) {
			continue
		}
		for _, right := range candidates {
			evidence, ok := scorePair(left, right, cfg)
			if !ok {
				continue
			}
			scoredByPrimary[left.ID] = append(scoredByPrimary[left.ID], evidence)
		}
	}

	results := make([]Evidence, 0, len(scoredByPrimary))
	for _, scored := range scoredByPrimary {
		sort.Slice(scored, func(i, j int) bool {
			if scored[i].Confidence == scored[j].Confidence {
				return scored[i].CandidateTrackID < scored[j].CandidateTrackID
			}
			return scored[i].Confidence > scored[j].Confidence
		})
		best := scored[0]
		if len(scored) > 1 && best.Confidence-scored[1].Confidence <= cfg.AmbiguityMargin {
			best.Status = "ambiguous"
			best.Reasons = append(best.Reasons, "ambiguous_with="+scored[1].CandidateTrackID)
		}
		results = append(results, best)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].PrimaryTrackID == results[j].PrimaryTrackID {
			return results[i].CandidateTrackID < results[j].CandidateTrackID
		}
		return results[i].PrimaryTrackID < results[j].PrimaryTrackID
	})
	return results
}

func scorePair(left, right TrackObservation, cfg Config) (Evidence, bool) {
	if !validTrack(right) || left.ID == right.ID || strings.EqualFold(left.Source, right.Source) {
		return Evidence{}, false
	}
	distance := haversineMeters(left.Position, right.Position)
	if distance > cfg.MaxDistanceMeters {
		return Evidence{}, false
	}
	delta := absDuration(left.ObservedAt.Sub(right.ObservedAt))
	if delta > cfg.MaxTimeDelta {
		return Evidence{}, false
	}

	distanceScore := 1 - (distance / cfg.MaxDistanceMeters)
	timeScore := 1 - (float64(delta) / float64(cfg.MaxTimeDelta))
	sourceScore := normalizedSourceConfidence(left.Confidence, right.Confidence)
	confidence := roundConfidence((distanceScore * 0.55) + (timeScore * 0.30) + (sourceScore * 0.15))
	if confidence < cfg.MinConfidence {
		return Evidence{}, false
	}

	observedAt := left.ObservedAt
	if right.ObservedAt.After(observedAt) {
		observedAt = right.ObservedAt
	}
	return Evidence{
		EntityID:             EntityID(cfg.Org, cfg.Platform, left.ID, right.ID),
		PrimaryTrackID:       left.ID,
		CandidateTrackID:     right.ID,
		Status:               "associated",
		Confidence:           confidence,
		Algorithm:            cfg.Algorithm,
		DistanceMeters:       math.Round(distance),
		TimeDelta:            delta,
		ObservedAt:           observedAt,
		Reasons:              associationReasons(left, right, distance, delta),
		PrimarySourceRef:     left.SourceRef,
		CandidateSourceRef:   right.SourceRef,
		PrimaryTrackNative:   left.NativeID,
		CandidateTrackNative: right.NativeID,
	}, true
}

func EntityID(org, platform, primaryTrackID, candidateTrackID string) string {
	primary := trackEntityToken(primaryTrackID)
	candidate := trackEntityToken(candidateTrackID)
	if candidate < primary {
		primary, candidate = candidate, primary
	}
	return fmt.Sprintf("%s.%s.cop.fusion.%s.%s-to-%s",
		entityToken(org),
		entityToken(platform),
		cop.EntityAssociation,
		primary,
		candidate,
	)
}

func trackEntityToken(trackID string) string {
	parts := strings.Split(strings.TrimSpace(trackID), ".")
	if len(parts) == 6 {
		return entityInstanceToken(parts[3] + "-" + parts[5])
	}
	return entityInstanceToken(trackID)
}

func entityInstanceToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == '.' || r == '-' || r == '_':
			if builder.Len() > 0 && !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		default:
			if builder.Len() > 0 && !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "unknown"
	}
	return token
}

func associationReasons(left, right TrackObservation, distance float64, delta time.Duration) []string {
	return []string{
		fmt.Sprintf("sources=%s,%s", nonEmpty(left.Source, "unknown"), nonEmpty(right.Source, "unknown")),
		fmt.Sprintf("distance_meters=%.0f", math.Round(distance)),
		fmt.Sprintf("time_delta_seconds=%.3f", delta.Seconds()),
	}
}

func normalizeConfig(cfg Config) Config {
	if cfg.Org == "" {
		cfg.Org = "c360"
	}
	if cfg.Platform == "" {
		cfg.Platform = "edge"
	}
	if cfg.Algorithm == "" {
		cfg.Algorithm = DefaultAlgorithm
	}
	if cfg.MaxDistanceMeters <= 0 {
		cfg.MaxDistanceMeters = DefaultMaxDistanceMeters
	}
	if cfg.MaxTimeDelta <= 0 {
		cfg.MaxTimeDelta = DefaultMaxTimeDelta
	}
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = DefaultMinConfidence
	}
	if cfg.AmbiguityMargin <= 0 {
		cfg.AmbiguityMargin = DefaultAmbiguityMargin
	}
	return cfg
}

func validTrack(track TrackObservation) bool {
	return track.ID != "" &&
		track.ObservedAt.IsZero() == false &&
		track.Position.Lat >= -90 && track.Position.Lat <= 90 &&
		track.Position.Lon >= -180 && track.Position.Lon <= 180
}

func normalizedSourceConfidence(left, right float64) float64 {
	left = clamp01(left)
	right = clamp01(right)
	if left == 0 && right == 0 {
		return 0.5
	}
	return (left + right) / 2
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

func roundConfidence(value float64) float64 {
	return math.Round(clamp01(value)*1000) / 1000
}

func absDuration(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}

func nonEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func haversineMeters(a, b GeoPoint) float64 {
	const earthRadiusMeters = 6371000
	lat1 := degreesToRadians(a.Lat)
	lat2 := degreesToRadians(b.Lat)
	deltaLat := degreesToRadians(b.Lat - a.Lat)
	deltaLon := degreesToRadians(b.Lon - a.Lon)

	sinLat := math.Sin(deltaLat / 2)
	sinLon := math.Sin(deltaLon / 2)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}

func degreesToRadians(value float64) float64 {
	return value * math.Pi / 180
}

func entityToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == '.' || r == '-':
			if builder.Len() > 0 && !lastDash {
				builder.WriteRune(r)
				lastDash = r == '-'
			}
		default:
			if builder.Len() > 0 && !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}
	token := strings.Trim(builder.String(), ".-")
	if token == "" {
		return "unknown"
	}
	return token
}
