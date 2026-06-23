package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
)

type NativeStatusEvidence struct {
	NativeID     string
	Protocol     string
	NativeStatus string
	Detail       string
	ObservedAt   time.Time
	Source       string
	SourceRef    string
}

type StatusUpdate struct {
	NativeID    string
	Status      string
	Description string
	ObservedAt  time.Time
	Source      string
	SourceRef   string
}

func ReconcileNativeStatus(current Intent, evidence NativeStatusEvidence) (StatusUpdate, error) {
	if err := current.validate(); err != nil {
		return StatusUpdate{}, fmt.Errorf("current command intent: %w", err)
	}
	if evidence.NativeID != "" && strings.TrimSpace(evidence.NativeID) != strings.TrimSpace(current.NativeID) {
		return StatusUpdate{}, fmt.Errorf("native status evidence native_id %q does not match current command %q", evidence.NativeID, current.NativeID)
	}
	if strings.TrimSpace(evidence.Protocol) == "" {
		return StatusUpdate{}, fmt.Errorf("native status evidence protocol is required")
	}
	if strings.TrimSpace(evidence.NativeStatus) == "" {
		return StatusUpdate{}, fmt.Errorf("native status evidence status is required")
	}
	if evidence.ObservedAt.IsZero() {
		return StatusUpdate{}, fmt.Errorf("native status evidence observed_at is required")
	}

	status, err := lifecycleStatusFromNative(evidence.Protocol, evidence.NativeStatus)
	if err != nil {
		return StatusUpdate{}, err
	}
	if err := ValidateStatusTransition(current.Status, status); err != nil {
		return StatusUpdate{}, fmt.Errorf("reconcile native command status: %w", err)
	}

	protocol := normalizeAuthority(evidence.Protocol)
	return StatusUpdate{
		NativeID:    strings.TrimSpace(current.NativeID),
		Status:      status,
		Description: nativeStatusDescription(protocol, evidence.NativeStatus, evidence.Detail),
		ObservedAt:  evidence.ObservedAt.UTC(),
		Source:      firstNonEmpty(evidence.Source, "command.reconcile."+protocol),
		SourceRef:   strings.TrimSpace(evidence.SourceRef),
	}, nil
}

func (p *Projector) ProjectStatusUpdate(update StatusUpdate) (Plan, error) {
	if p == nil {
		return Plan{}, fmt.Errorf("command intent projector is nil")
	}
	if strings.TrimSpace(update.NativeID) == "" {
		return Plan{}, fmt.Errorf("command status native_id is required")
	}
	if err := validateStatus(update.Status); err != nil {
		return Plan{}, err
	}
	if update.ObservedAt.IsZero() {
		return Plan{}, fmt.Errorf("command status observed_at is required")
	}

	intentID := p.intentID(update.NativeID)
	triples := []message.Triple{
		p.statusTriple(intentID, cop.TaskNativeID, strings.TrimSpace(update.NativeID), update),
		p.statusTriple(intentID, cop.TaskStatus, statusOrDefault(update.Status), update),
		p.statusTriple(intentID, cop.TaskDescription, strings.TrimSpace(update.Description), update),
		p.statusTriple(intentID, cop.ProvenanceSource, sourceFromStatusUpdate(update), update),
		p.statusTriple(intentID, cop.ProvenanceConfidence, p.cfg.Confidence, update),
		p.statusTriple(intentID, cop.ProvenanceObservedAt, update.ObservedAt.UTC(), update),
	}
	if strings.TrimSpace(update.SourceRef) != "" {
		triples = append(triples, p.statusTriple(intentID, cop.ProvenanceSourceRef, strings.TrimSpace(update.SourceRef), update))
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
			RequestID:       requestID("update-command-status", update.NativeID),
		},
	}}}, nil
}

func (p *Projector) statusTriple(subject string, predicate string, object any, update StatusUpdate) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     sourceFromStatusUpdate(update),
		Timestamp:  update.ObservedAt.UTC(),
		Confidence: p.cfg.Confidence,
	}
}

func lifecycleStatusFromNative(protocol string, nativeStatus string) (string, error) {
	normalizedProtocol := normalizeAuthority(protocol)
	normalizedStatus := normalizeStatus(nativeStatus)
	if normalizedProtocol == "mavlink" || normalizedProtocol == "mavlink.command_ack" {
		switch normalizedStatus {
		case "accepted":
			return StatusAccepted, nil
		case "in_progress":
			return StatusExecuting, nil
		case "cancelled":
			return StatusCancelled, nil
		case "failed":
			return StatusFailed, nil
		case "temporarily_rejected", "denied", "unsupported":
			return StatusRejected, nil
		default:
			return "", fmt.Errorf("unsupported MAVLink command status %q", nativeStatus)
		}
	}
	if err := validateStatus(normalizedStatus); err != nil {
		return "", fmt.Errorf("unsupported native command status %q for protocol %q", nativeStatus, protocol)
	}
	return normalizedStatus, nil
}

func nativeStatusDescription(protocol string, nativeStatus string, detail string) string {
	detail = strings.TrimSpace(detail)
	description := fmt.Sprintf("native %s status: %s", protocol, normalizeStatus(nativeStatus))
	if detail == "" {
		return description
	}
	return description + " - " + detail
}

func sourceFromStatusUpdate(update StatusUpdate) string {
	if strings.TrimSpace(update.Source) == "" {
		return "command.reconcile"
	}
	return strings.TrimSpace(update.Source)
}
