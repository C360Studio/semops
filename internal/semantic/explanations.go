// Package semantic derives read-only COP explanation artifacts from governed
// snapshot state.
package semantic

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	copapi "github.com/c360studio/semops/internal/api/cop"
	"github.com/c360studio/semops/pkg/cop"
)

const (
	AlgorithmDeterministicV1 = "semops.semantic.deterministic.v1"

	KindTrackTranslation       = "semantic.track_translation"
	KindAssociationExplanation = "semantic.association_explanation"
	KindAssociationAnomaly     = "semantic.association_anomaly"

	TaskTrackTranslation   = "translate_track_state"
	TaskAssociationAnomaly = "explain_association_anomaly"

	ClaimPostureReadOnly = "read-only derived explanation; no command authority; no source mutation; no identity merge"
)

type ExplanationSet struct {
	GeneratedAt  time.Time     `json:"generated_at"`
	Algorithm    string        `json:"algorithm"`
	ClaimPosture string        `json:"claim_posture"`
	Items        []Explanation `json:"items"`
}

type Explanation struct {
	ID            string        `json:"id"`
	Kind          string        `json:"kind"`
	EntityID      string        `json:"entity_id"`
	EntityType    string        `json:"entity_type"`
	Title         string        `json:"title"`
	Output        string        `json:"output"`
	Severity      string        `json:"severity"`
	Status        string        `json:"status"`
	Task          Task          `json:"task"`
	TrajectoryRef string        `json:"trajectory_ref"`
	Evidence      []EvidenceRef `json:"evidence"`
	GeneratedAt   time.Time     `json:"generated_at"`
	Algorithm     string        `json:"algorithm"`
	ClaimPosture  string        `json:"claim_posture"`
}

type Task struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Prompt   string `json:"prompt"`
	InputRef string `json:"input_ref"`
}

type EvidenceRef struct {
	EntityID   string    `json:"entity_id"`
	EntityType string    `json:"entity_type"`
	Role       string    `json:"role"`
	Source     string    `json:"source"`
	Owner      string    `json:"owner"`
	SourceRef  string    `json:"source_ref"`
	ObservedAt time.Time `json:"observed_at"`
	Confidence float64   `json:"confidence"`
}

// Build returns deterministic semantic explanations for the read-only COP
// snapshot. It records the task, source evidence, output, and trajectory
// reference needed before semantic results can be promoted into an operator UI.
func Build(snapshot copapi.Snapshot) ExplanationSet {
	generatedAt := snapshot.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = latestObservedAt(snapshot)
	}

	tracks := sortedTracks(snapshot.Tracks)
	trackByID := make(map[string]copapi.Track, len(tracks))
	for _, track := range tracks {
		trackByID[track.ID] = track
	}
	alertsByEntity := alertsByEntity(snapshot.Alerts)

	items := make([]Explanation, 0, len(snapshot.Tracks)+len(snapshot.Associations))
	for _, track := range tracks {
		items = append(items, trackTranslation(snapshot.Scenario, generatedAt, track, alertsByEntity[track.ID]))
	}
	for _, association := range sortedAssociations(snapshot.Associations) {
		items = append(items, associationExplanation(snapshot.Scenario, generatedAt, association, trackByID))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return ExplanationSet{
		GeneratedAt:  generatedAt,
		Algorithm:    AlgorithmDeterministicV1,
		ClaimPosture: ClaimPostureReadOnly,
		Items:        items,
	}
}

func trackTranslation(scenario string, generatedAt time.Time, track copapi.Track, alerts []copapi.Alert) Explanation {
	inputRef := snapshotEntityRef(scenario, "tracks", track.ID)
	trajectoryRef := semanticTrajectoryRef(scenario, KindTrackTranslation, track.ID)
	sourceRef := nonEmpty(track.Provenance.SourceRef, track.ID)
	alertText := ""
	if len(alerts) > 0 {
		alert := alerts[0]
		alertText = fmt.Sprintf(" Active alert %s is %s: %s.", label(alert.Label, alert.ID), alert.Severity, alert.Reason)
	}
	velocityText := ""
	if strings.TrimSpace(track.Velocity) != "" {
		velocityText = fmt.Sprintf(" Velocity is %s.", track.Velocity)
	}

	return Explanation{
		ID:         semanticID(KindTrackTranslation, track.ID),
		Kind:       KindTrackTranslation,
		EntityID:   track.ID,
		EntityType: cop.EntityTrack,
		Title:      fmt.Sprintf("%s track state", label(track.Label, track.ID)),
		Output: fmt.Sprintf(
			"%s from %s is %s at %.4f, %.4f with confidence %.2f. Observed %.0f seconds before the snapshot using source reference %s.%s%s",
			label(track.Label, track.ID),
			nonEmpty(track.Source, "unknown"),
			nonEmpty(track.Status, "unknown"),
			track.Position.Lat,
			track.Position.Lon,
			track.Confidence,
			ageSeconds(generatedAt, track.UpdatedAt),
			sourceRef,
			velocityText,
			alertText,
		),
		Severity:      trackSeverity(track, alerts),
		Status:        "translated",
		Task:          trackTask(track.ID, inputRef),
		TrajectoryRef: trajectoryRef,
		Evidence:      trackEvidence(track, "source_track", cop.EntityTrack, alerts),
		GeneratedAt:   generatedAt,
		Algorithm:     AlgorithmDeterministicV1,
		ClaimPosture:  ClaimPostureReadOnly,
	}
}

func associationExplanation(
	scenario string,
	generatedAt time.Time,
	association copapi.Association,
	trackByID map[string]copapi.Track,
) Explanation {
	inputRef := snapshotEntityRef(scenario, "associations", association.ID)
	trajectoryRef := semanticTrajectoryRef(scenario, KindAssociationAnomaly, association.ID)
	kind := KindAssociationExplanation
	severity := "info"
	status := "explained"
	if associationNeedsReview(association) {
		kind = KindAssociationAnomaly
		severity = "watch"
		status = "needs_review"
	} else {
		trajectoryRef = semanticTrajectoryRef(scenario, KindAssociationExplanation, association.ID)
	}

	primaryLabel := association.PrimaryTrackID
	if track, ok := trackByID[association.PrimaryTrackID]; ok {
		primaryLabel = label(track.Label, track.ID)
	}
	candidateLabel := association.CandidateTrackID
	if track, ok := trackByID[association.CandidateTrackID]; ok {
		candidateLabel = label(track.Label, track.ID)
	}

	return Explanation{
		ID:         semanticID(kind, association.ID),
		Kind:       kind,
		EntityID:   association.ID,
		EntityType: cop.EntityAssociation,
		Title:      fmt.Sprintf("%s association evidence", label(association.Label, association.ID)),
		Output: fmt.Sprintf(
			"%s links %s and %s with confidence %.2f using %s. %s%s Claim posture: %s. Source tracks remain separate.",
			label(association.Label, association.ID),
			primaryLabel,
			candidateLabel,
			association.Confidence,
			nonEmpty(association.Algorithm, "unknown algorithm"),
			associationMetricText(association),
			reasonText(association.Reason),
			nonEmpty(association.ClaimPosture, ClaimPostureReadOnly),
		),
		Severity:      severity,
		Status:        status,
		Task:          associationTask(association.ID, inputRef),
		TrajectoryRef: trajectoryRef,
		Evidence:      associationEvidence(association, trackByID),
		GeneratedAt:   generatedAt,
		Algorithm:     AlgorithmDeterministicV1,
		ClaimPosture:  ClaimPostureReadOnly,
	}
}

func trackTask(trackID, inputRef string) Task {
	return Task{
		ID:       "semantic.task.track_translation." + entityToken(trackID),
		Kind:     TaskTrackTranslation,
		Prompt:   "Translate source-owned COP track state into an operator-readable status explanation without changing source state.",
		InputRef: inputRef,
	}
}

func associationTask(associationID, inputRef string) Task {
	return Task{
		ID:       "semantic.task.association_anomaly." + entityToken(associationID),
		Kind:     TaskAssociationAnomaly,
		Prompt:   "Explain why fusion association evidence is review-worthy while preserving source-owned track state.",
		InputRef: inputRef,
	}
}

func trackEvidence(track copapi.Track, role, entityType string, alerts []copapi.Alert) []EvidenceRef {
	evidence := []EvidenceRef{
		{
			EntityID:   track.ID,
			EntityType: entityType,
			Role:       role,
			Source:     nonEmpty(track.Source, "unknown"),
			Owner:      nonEmpty(track.Provenance.Owner, "unknown"),
			SourceRef:  nonEmpty(track.Provenance.SourceRef, track.ID),
			ObservedAt: firstNonZeroTime(track.Provenance.Observed, track.UpdatedAt),
			Confidence: track.Confidence,
		},
	}
	for _, alert := range sortedAlerts(alerts) {
		evidence = append(evidence, EvidenceRef{
			EntityID:   alert.ID,
			EntityType: cop.EntityAlert,
			Role:       "alert",
			Source:     "cop-alert",
			Owner:      "semops.cop.alerts",
			SourceRef:  alert.ID,
			ObservedAt: alert.UpdatedAt,
			Confidence: 1,
		})
	}
	return evidence
}

func associationEvidence(association copapi.Association, trackByID map[string]copapi.Track) []EvidenceRef {
	evidence := []EvidenceRef{
		{
			EntityID:   association.ID,
			EntityType: cop.EntityAssociation,
			Role:       "association",
			Source:     nonEmpty(association.Source, "fusion"),
			Owner:      nonEmpty(association.Provenance.Owner, cop.OwnerFusion),
			SourceRef:  nonEmpty(association.Provenance.SourceRef, association.ID),
			ObservedAt: firstNonZeroTime(association.Provenance.Observed, association.UpdatedAt),
			Confidence: association.Confidence,
		},
	}
	if track, ok := trackByID[association.PrimaryTrackID]; ok {
		evidence = append(evidence, trackEvidence(track, "primary_track", cop.EntityTrack, nil)[0])
	}
	if track, ok := trackByID[association.CandidateTrackID]; ok {
		evidence = append(evidence, trackEvidence(track, "candidate_track", cop.EntityTrack, nil)[0])
	}
	return evidence
}

func associationNeedsReview(association copapi.Association) bool {
	posture := strings.ToLower(association.ClaimPosture + " " + association.Status)
	if strings.Contains(posture, "ambiguous") ||
		strings.Contains(posture, "candidate") ||
		strings.Contains(posture, "review") ||
		strings.Contains(posture, "challenged") {
		return true
	}
	return association.Confidence > 0 && association.Confidence < 0.85
}

func associationMetricText(association copapi.Association) string {
	parts := make([]string, 0, 2)
	if association.DistanceMeters != nil {
		parts = append(parts, fmt.Sprintf("distance %.0f m", *association.DistanceMeters))
	}
	if association.TimeDeltaSeconds != nil {
		parts = append(parts, fmt.Sprintf("time delta %.2f s", *association.TimeDeltaSeconds))
	}
	if len(parts) == 0 {
		return "No distance or time-delta metric was supplied."
	}
	return "The supporting metrics are " + strings.Join(parts, " and ") + "."
}

func reasonText(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ""
	}
	return " Reason: " + reason + "."
}

func trackSeverity(track copapi.Track, alerts []copapi.Alert) string {
	for _, alert := range alerts {
		switch strings.ToLower(alert.Severity) {
		case "critical", "warning", "watch":
			return alert.Severity
		}
	}
	if track.Confidence > 0 && track.Confidence < 0.75 {
		return "watch"
	}
	return "info"
}

func sortedTracks(tracks []copapi.Track) []copapi.Track {
	out := append([]copapi.Track(nil), tracks...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func sortedAssociations(associations []copapi.Association) []copapi.Association {
	out := append([]copapi.Association(nil), associations...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func sortedAlerts(alerts []copapi.Alert) []copapi.Alert {
	out := append([]copapi.Alert(nil), alerts...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func alertsByEntity(alerts []copapi.Alert) map[string][]copapi.Alert {
	byEntity := make(map[string][]copapi.Alert)
	for _, alert := range alerts {
		if strings.TrimSpace(alert.EntityID) == "" {
			continue
		}
		byEntity[alert.EntityID] = append(byEntity[alert.EntityID], alert)
	}
	return byEntity
}

func snapshotEntityRef(scenario, collection, entityID string) string {
	return "cop://snapshot/" +
		url.PathEscape(nonEmpty(scenario, "unknown")) +
		"/" +
		url.PathEscape(collection) +
		"/" +
		url.PathEscape(entityID)
}

func semanticTrajectoryRef(scenario, kind, entityID string) string {
	return "semops://semantic/trajectory/" +
		url.PathEscape(nonEmpty(scenario, "unknown")) +
		"/" +
		url.PathEscape(entityToken(kind)) +
		"/" +
		url.PathEscape(entityToken(entityID))
}

func semanticID(kind, entityID string) string {
	return "semops." + entityToken(kind) + "." + entityToken(entityID)
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

func latestObservedAt(snapshot copapi.Snapshot) time.Time {
	var latest time.Time
	for _, track := range snapshot.Tracks {
		latest = laterTime(latest, track.UpdatedAt)
	}
	for _, association := range snapshot.Associations {
		latest = laterTime(latest, association.UpdatedAt)
	}
	for _, alert := range snapshot.Alerts {
		latest = laterTime(latest, alert.UpdatedAt)
	}
	return latest
}

func laterTime(left, right time.Time) time.Time {
	if right.After(left) {
		return right
	}
	return left
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func ageSeconds(reference, observed time.Time) float64 {
	if reference.IsZero() || observed.IsZero() {
		return 0
	}
	age := reference.Sub(observed).Seconds()
	if age < 0 {
		return 0
	}
	return age
}

func label(preferred, fallback string) string {
	return nonEmpty(preferred, fallback)
}

func nonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
