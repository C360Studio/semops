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
	cfg                    Config
	bornAssociations       map[string]struct{}
	bornAssociationReviews map[string]struct{}
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
		cfg:                    cfg,
		bornAssociations:       make(map[string]struct{}),
		bornAssociationReviews: make(map[string]struct{}),
	}
}

type AssociationReviewEvidence struct {
	Org           string
	Platform      string
	AssociationID string
	Decision      string
	ReviewedBy    string
	ReviewedAt    time.Time
	Comment       string
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

func (p *Projector) ProjectAssociationReview(evidence AssociationReviewEvidence) (Plan, error) {
	return p.projectAssociationReview(evidence, cloneStringSet(p.bornAssociationReviews))
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

func (p *Projector) projectAssociationReview(
	evidence AssociationReviewEvidence,
	born map[string]struct{},
) (Plan, error) {
	if p == nil {
		return Plan{}, fmt.Errorf("fusion projector is nil")
	}
	if err := validateAssociationReview(evidence); err != nil {
		return Plan{}, err
	}
	entityID := AssociationReviewEntityID(evidence.Org, evidence.Platform, evidence.AssociationID)
	createTriples := p.associationReviewTriples(entityID, evidence, true)
	if _, ok := born[entityID]; !ok {
		born[entityID] = struct{}{}
		return Plan{Mutations: []Mutation{{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          entityID,
					MessageType: messageType(cop.FusionAssociationReviewContract().MessageType),
					UpdatedAt:   evidence.ReviewedAt.UTC(),
				},
				Triples:         createTriples,
				IndexingProfile: cop.FusionAssociationReviewContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerFusion),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-association-review", entityID),
			},
		}}}, nil
	}

	return Plan{Mutations: []Mutation{{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: entityID,
			},
			AddTriples:      p.associationReviewTriples(entityID, evidence, false),
			IndexingProfile: cop.FusionAssociationReviewContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerFusion),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("update-association-review", entityID),
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

func (p *Projector) MarkBornForAssociationReview(evidence AssociationReviewEvidence, entityID string) bool {
	if p == nil || entityID == "" || entityID != AssociationReviewEntityID(evidence.Org, evidence.Platform, evidence.AssociationID) {
		return false
	}
	p.bornAssociationReviews[entityID] = struct{}{}
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

func (p *Projector) associationReviewTriples(
	entityID string,
	evidence AssociationReviewEvidence,
	includeEdges bool,
) []message.Triple {
	when := evidence.ReviewedAt.UTC()
	triples := []message.Triple{
		p.reviewTriple(entityID, cop.AssociationReviewDecision, strings.ToLower(strings.TrimSpace(evidence.Decision)), evidence, when),
		p.reviewTriple(entityID, cop.AssociationReviewReviewedBy, evidence.ReviewedBy, evidence, when),
		p.reviewTriple(entityID, cop.AssociationReviewReviewedAt, when, evidence, when),
		p.reviewTriple(entityID, cop.AssociationReviewComment, evidence.Comment, evidence, when),
		p.reviewTriple(entityID, cop.ProvenanceSource, "operator.association_review", evidence, when),
		p.reviewTriple(entityID, cop.ProvenanceConfidence, 1.0, evidence, when),
		p.reviewTriple(entityID, cop.ProvenanceObservedAt, when, evidence, when),
		p.reviewTriple(entityID, cop.ProvenanceSourceRef, associationReviewSourceRef(evidence), evidence, when),
	}
	if includeEdges {
		triples = append(triples, p.reviewTriple(entityID, cop.AssociationReviewAssociation, evidence.AssociationID, evidence, when))
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

func (p *Projector) reviewTriple(
	subject string,
	predicate string,
	object any,
	evidence AssociationReviewEvidence,
	when time.Time,
) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "operator.association_review",
		Timestamp:  when.UTC(),
		Confidence: 1,
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

func validateAssociationReview(evidence AssociationReviewEvidence) error {
	switch {
	case strings.TrimSpace(evidence.Org) == "":
		return fmt.Errorf("fusion association review org is required")
	case strings.TrimSpace(evidence.Platform) == "":
		return fmt.Errorf("fusion association review platform is required")
	case strings.TrimSpace(evidence.AssociationID) == "":
		return fmt.Errorf("fusion association review association id is required")
	case strings.TrimSpace(evidence.Decision) == "":
		return fmt.Errorf("fusion association review decision is required")
	case strings.TrimSpace(evidence.ReviewedBy) == "":
		return fmt.Errorf("fusion association review reviewer is required")
	case evidence.ReviewedAt.IsZero():
		return fmt.Errorf("fusion association review reviewed_at is required")
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

func associationReviewSourceRef(evidence AssociationReviewEvidence) string {
	parts := []string{"association=" + evidence.AssociationID}
	if evidence.ReviewedBy != "" {
		parts = append(parts, "reviewer="+evidence.ReviewedBy)
	}
	return strings.Join(parts, " ")
}

func AssociationReviewEntityID(org, platform, associationID string) string {
	return strings.Join([]string{
		strings.TrimSpace(org),
		strings.TrimSpace(platform),
		"cop",
		"fusion",
		cop.EntityAssociationReview,
		associationReviewToken(associationID),
	}, ".")
}

func associationReviewToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
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
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return token
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
