// Package fusion projects derived COP fusion evidence into SemStreams graph
// mutation plans.
package fusion

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	fusionassociation "github.com/c360studio/semops/internal/fusion/association"
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
	OwnerTokens map[string]ownership.OwnerToken
	TraceID     string
}

type Projector struct {
	cfg              Config
	bornAssociations map[string]struct{}
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
	cfg.OwnerTokens = cloneOwnerTokens(cfg.OwnerTokens)
	return &Projector{
		cfg:              cfg,
		bornAssociations: make(map[string]struct{}),
	}
}

func (p *Projector) ProjectAssociation(evidence fusionassociation.Evidence) (Plan, error) {
	return p.projectAssociation(evidence, cloneStringSet(p.bornAssociations))
}

func (p *Projector) ProjectAssociations(evidence []fusionassociation.Evidence) (Plan, error) {
	born := cloneStringSet(p.bornAssociations)
	var plan Plan
	for _, item := range evidence {
		next, err := p.projectAssociation(item, born)
		if err != nil {
			return Plan{}, err
		}
		plan.Mutations = append(plan.Mutations, next.Mutations...)
	}
	return plan, nil
}

func (p *Projector) projectAssociation(
	evidence fusionassociation.Evidence,
	born map[string]struct{},
) (Plan, error) {
	if p == nil {
		return Plan{}, fmt.Errorf("fusion projector is nil")
	}
	if err := validateAssociation(evidence); err != nil {
		return Plan{}, err
	}

	createTriples := p.associationTriples(evidence, true)
	if _, ok := born[evidence.EntityID]; !ok {
		born[evidence.EntityID] = struct{}{}
		return Plan{Mutations: []Mutation{{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          evidence.EntityID,
					MessageType: messageType(cop.FusionTrackAssociationContract().MessageType),
					UpdatedAt:   evidence.ObservedAt.UTC(),
				},
				Triples:         createTriples,
				IndexingProfile: cop.FusionTrackAssociationContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerFusion),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-track-association", evidence.EntityID),
			},
		}}}, nil
	}

	return Plan{Mutations: []Mutation{{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: evidence.EntityID,
			},
			AddTriples:      p.associationTriples(evidence, false),
			IndexingProfile: cop.FusionTrackAssociationContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerFusion),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("update-track-association", evidence.EntityID),
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
		if _, ok := p.bornAssociations[mutation.Create.Entity.ID]; ok {
			continue
		}
		p.bornAssociations[mutation.Create.Entity.ID] = struct{}{}
		marked++
	}
	return marked
}

func (p *Projector) MarkBornForAssociation(evidence fusionassociation.Evidence, entityID string) bool {
	if p == nil || entityID == "" || entityID != evidence.EntityID {
		return false
	}
	p.bornAssociations[entityID] = struct{}{}
	return true
}

func (p *Projector) associationTriples(evidence fusionassociation.Evidence, includeEdges bool) []message.Triple {
	when := evidence.ObservedAt.UTC()
	triples := []message.Triple{
		p.triple(evidence.EntityID, cop.AssociationKind, "track", evidence, when),
		p.triple(evidence.EntityID, cop.AssociationStatus, evidence.Status, evidence, when),
		p.triple(evidence.EntityID, cop.AssociationConfidence, evidence.Confidence, evidence, when),
		p.triple(evidence.EntityID, cop.AssociationAlgorithm, evidence.Algorithm, evidence, when),
		p.triple(evidence.EntityID, cop.AssociationReason, strings.Join(evidence.Reasons, "; "), evidence, when),
		p.triple(evidence.EntityID, cop.AssociationDistanceMeters, evidence.DistanceMeters, evidence, when),
		p.triple(evidence.EntityID, cop.AssociationTimeDeltaSeconds, evidence.TimeDelta.Seconds(), evidence, when),
		p.triple(evidence.EntityID, cop.AssociationObservedAt, when, evidence, when),
		p.triple(evidence.EntityID, cop.ProvenanceSource, "fusion.track_association", evidence, when),
		p.triple(evidence.EntityID, cop.ProvenanceConfidence, evidence.Confidence, evidence, when),
		p.triple(evidence.EntityID, cop.ProvenanceObservedAt, when, evidence, when),
		p.triple(evidence.EntityID, cop.ProvenanceSourceRef, associationSourceRef(evidence), evidence, when),
	}
	if includeEdges {
		triples = append(triples,
			p.triple(evidence.EntityID, cop.AssociationPrimaryTrack, evidence.PrimaryTrackID, evidence, when),
			p.triple(evidence.EntityID, cop.AssociationCandidateTrack, evidence.CandidateTrackID, evidence, when),
		)
	}
	return triples
}

func (p *Projector) triple(
	subject string,
	predicate string,
	object any,
	evidence fusionassociation.Evidence,
	when time.Time,
) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "fusion.track_association",
		Timestamp:  when.UTC(),
		Confidence: evidence.Confidence,
	}
}

func validateAssociation(evidence fusionassociation.Evidence) error {
	switch {
	case strings.TrimSpace(evidence.EntityID) == "":
		return fmt.Errorf("fusion association entity id is required")
	case strings.TrimSpace(evidence.PrimaryTrackID) == "":
		return fmt.Errorf("fusion association primary track id is required")
	case strings.TrimSpace(evidence.CandidateTrackID) == "":
		return fmt.Errorf("fusion association candidate track id is required")
	case strings.TrimSpace(evidence.Status) == "":
		return fmt.Errorf("fusion association status is required")
	case strings.TrimSpace(evidence.Algorithm) == "":
		return fmt.Errorf("fusion association algorithm is required")
	case evidence.ObservedAt.IsZero():
		return fmt.Errorf("fusion association observed_at is required")
	case evidence.Confidence <= 0 || evidence.Confidence > 1:
		return fmt.Errorf("fusion association confidence must be within (0,1]")
	case evidence.DistanceMeters < 0:
		return fmt.Errorf("fusion association distance must be non-negative")
	case evidence.TimeDelta < 0:
		return fmt.Errorf("fusion association time delta must be non-negative")
	default:
		return nil
	}
}

func (p *Projector) ownerToken(owner string) string {
	if p == nil || p.cfg.OwnerTokens == nil {
		return ""
	}
	return p.cfg.OwnerTokens[owner].Wire()
}

func associationSourceRef(evidence fusionassociation.Evidence) string {
	parts := make([]string, 0, 2)
	if evidence.PrimarySourceRef != "" {
		parts = append(parts, "primary="+evidence.PrimarySourceRef)
	}
	if evidence.CandidateSourceRef != "" {
		parts = append(parts, "candidate="+evidence.CandidateSourceRef)
	}
	return strings.Join(parts, " ")
}

func requestID(prefix, entityID string) string {
	return prefix + "-" + entityToken(entityID)
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

func cloneStringSet(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for key := range in {
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

func entityToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case r == '.':
			builder.WriteRune('.')
			lastDash = false
		case r == '-':
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
	token := strings.Trim(builder.String(), ".-")
	if token == "" {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return token
}
