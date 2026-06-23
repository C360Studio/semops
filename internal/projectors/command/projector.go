package command

import (
	"fmt"
	"strings"
	"time"
	"unicode"

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

type Intent struct {
	NativeID       string
	TargetAssetID  string
	Name           string
	Kind           string
	Status         string
	Description    string
	DesiredState   string
	Authority      string
	Priority       int
	ExpiresAt      time.Time
	CorrelationID  string
	IdempotencyKey string
	RequestedBy    string
	ObservedAt     time.Time
	Source         string
	SourceRef      string
}

type Projector struct {
	cfg         Config
	bornIntents map[string]struct{}
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
		cfg.Confidence = 1.0
	}
	cfg.OwnerTokens = cloneOwnerTokens(cfg.OwnerTokens)
	return &Projector{
		cfg:         cfg,
		bornIntents: make(map[string]struct{}),
	}
}

func (p *Projector) ProjectIntent(intent Intent) (Plan, error) {
	return p.projectIntent(intent, cloneStringSet(p.bornIntents))
}

func (p *Projector) ProjectIntents(intents []Intent) (Plan, error) {
	bornIntents := cloneStringSet(p.bornIntents)
	var plan Plan
	for _, intent := range intents {
		next, err := p.projectIntent(intent, bornIntents)
		if err != nil {
			return Plan{}, err
		}
		plan.Mutations = append(plan.Mutations, next.Mutations...)
	}
	return plan, nil
}

func (p *Projector) projectIntent(intent Intent, bornIntents map[string]struct{}) (Plan, error) {
	if p == nil {
		return Plan{}, fmt.Errorf("command intent projector is nil")
	}
	if err := intent.validate(); err != nil {
		return Plan{}, err
	}

	intentID := p.intentID(intent.NativeID)
	triples := p.intentTriples(intentID, intent)
	if _, ok := bornIntents[intentID]; !ok {
		bornIntents[intentID] = struct{}{}
		triples = append(triples, p.triple(intentID, cop.TaskTarget, intent.TargetAssetID, intent))
		return Plan{Mutations: []Mutation{{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          intentID,
					MessageType: messageType(cop.CommandIntentContract().MessageType),
					UpdatedAt:   observedAt(intent),
				},
				Triples:         triples,
				IndexingProfile: cop.CommandIntentContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerCommand),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-command-intent", intent.NativeID),
			},
		}}}, nil
	}

	return Plan{Mutations: []Mutation{{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: intentID,
			},
			AddTriples:      triples,
			IndexingProfile: cop.CommandIntentContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerCommand),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("update-command-intent", intent.NativeID),
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
		if mutation.Create.Entity.MessageType.Key() != cop.CommandIntentContract().MessageType {
			continue
		}
		if _, ok := p.bornIntents[mutation.Create.Entity.ID]; ok {
			continue
		}
		p.bornIntents[mutation.Create.Entity.ID] = struct{}{}
		marked++
	}
	return marked
}

func (p *Projector) MarkBornForIntent(intent Intent, entityID string) bool {
	if p == nil || entityID == "" || entityID != p.intentID(intent.NativeID) {
		return false
	}
	p.bornIntents[entityID] = struct{}{}
	return true
}

func (p *Projector) intentTriples(intentID string, intent Intent) []message.Triple {
	triples := []message.Triple{
		p.triple(intentID, cop.TaskNativeID, strings.TrimSpace(intent.NativeID), intent),
		p.triple(intentID, cop.TaskName, firstNonEmpty(intent.Name, intent.NativeID), intent),
		p.triple(intentID, cop.TaskKind, strings.TrimSpace(intent.Kind), intent),
		p.triple(intentID, cop.TaskStatus, firstNonEmpty(intent.Status, "requested"), intent),
		p.triple(intentID, cop.TaskDesired, strings.TrimSpace(intent.DesiredState), intent),
		p.triple(intentID, cop.TaskAuthority, strings.TrimSpace(intent.Authority), intent),
		p.triple(intentID, cop.TaskPriority, int64(intent.Priority), intent),
		p.triple(intentID, cop.TaskExpiresAt, intent.ExpiresAt.UTC(), intent),
		p.triple(intentID, cop.TaskCorrelation, strings.TrimSpace(intent.CorrelationID), intent),
		p.triple(intentID, cop.TaskIdempotency, strings.TrimSpace(intent.IdempotencyKey), intent),
		p.triple(intentID, cop.TaskRequestedBy, strings.TrimSpace(intent.RequestedBy), intent),
		p.triple(intentID, cop.ProvenanceSource, source(intent), intent),
		p.triple(intentID, cop.ProvenanceConfidence, p.cfg.Confidence, intent),
		p.triple(intentID, cop.ProvenanceObservedAt, observedAt(intent), intent),
	}
	if strings.TrimSpace(intent.Description) != "" {
		triples = append(triples, p.triple(intentID, cop.TaskDescription, strings.TrimSpace(intent.Description), intent))
	}
	if strings.TrimSpace(intent.SourceRef) != "" {
		triples = append(triples, p.triple(intentID, cop.ProvenanceSourceRef, strings.TrimSpace(intent.SourceRef), intent))
	}
	return triples
}

func (p *Projector) triple(subject string, predicate string, object any, intent Intent) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     source(intent),
		Timestamp:  observedAt(intent),
		Confidence: p.cfg.Confidence,
	}
}

func (p *Projector) intentID(nativeID string) string {
	return EntityID(p.cfg.Org, p.cfg.Platform, nativeID)
}

func EntityID(org, platform, nativeID string) string {
	return strings.Join([]string{
		entityToken(org),
		entityToken(platform),
		"cop",
		"command",
		cop.EntityTask,
		entityToken(nativeID),
	}, ".")
}

func (p *Projector) ownerToken(owner string) string {
	if p == nil || p.cfg.OwnerTokens == nil {
		return ""
	}
	return p.cfg.OwnerTokens[owner].Wire()
}

func (i Intent) validate() error {
	if strings.TrimSpace(i.NativeID) == "" {
		return fmt.Errorf("command intent native_id is required")
	}
	if strings.TrimSpace(i.TargetAssetID) == "" {
		return fmt.Errorf("command intent target asset id is required")
	}
	if strings.TrimSpace(i.Kind) == "" {
		return fmt.Errorf("command intent kind is required")
	}
	if strings.TrimSpace(i.DesiredState) == "" {
		return fmt.Errorf("command intent desired_state is required")
	}
	if strings.TrimSpace(i.Authority) == "" {
		return fmt.Errorf("command intent authority is required")
	}
	if i.Priority < 1 || i.Priority > 100 {
		return fmt.Errorf("command intent priority must be between 1 and 100")
	}
	if i.ExpiresAt.IsZero() {
		return fmt.Errorf("command intent expires_at is required")
	}
	if !i.ObservedAt.IsZero() && !i.ExpiresAt.After(i.ObservedAt.UTC()) {
		return fmt.Errorf("command intent expires_at must be after observed_at")
	}
	if strings.TrimSpace(i.CorrelationID) == "" {
		return fmt.Errorf("command intent correlation_id is required")
	}
	if strings.TrimSpace(i.IdempotencyKey) == "" {
		return fmt.Errorf("command intent idempotency_key is required")
	}
	if strings.TrimSpace(i.RequestedBy) == "" {
		return fmt.Errorf("command intent requested_by is required")
	}
	return nil
}

func observedAt(intent Intent) time.Time {
	if !intent.ObservedAt.IsZero() {
		return intent.ObservedAt.UTC()
	}
	return time.Now().UTC()
}

func source(intent Intent) string {
	if strings.TrimSpace(intent.Source) == "" {
		return "command"
	}
	return strings.TrimSpace(intent.Source)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
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
