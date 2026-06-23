package command

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type ReplayKind string

const (
	ReplayIntent       ReplayKind = "intent"
	ReplayNativeStatus ReplayKind = "native_status"
	ReplayCancellation ReplayKind = "cancellation"
	ReplayDeadline     ReplayKind = "deadline"
)

type LifecycleReplayRecord struct {
	Ref          string
	Source       string
	Kind         ReplayKind
	Intent       *Intent
	NativeStatus *NativeStatusEvidence
	Cancellation *CancellationRequest
	Deadline     *DeadlineEvidence
}

func LifecycleFixtureRecords() []LifecycleReplayRecord {
	start := time.Date(2026, 6, 23, 20, 0, 0, 0, time.UTC)
	target := "c360.edge.cop.mavlink.asset.system-42"

	route := Intent{
		NativeID:       "csapi-command-route-42",
		TargetAssetID:  target,
		Name:           "Route MAVLink system 42 to North Gate",
		Kind:           "mavlink.goto",
		Status:         StatusRequested,
		Description:    "operator-approved route change",
		DesiredState:   `{"command":"goto","lat":38.9,"lon":-77.04}`,
		Authority:      "local.operator",
		Priority:       80,
		ExpiresAt:      start.Add(5 * time.Minute),
		CorrelationID:  "csapi:req-route-42",
		IdempotencyKey: "idem-route-42",
		RequestedBy:    "operator:coby",
		ObservedAt:     start,
		Source:         "cs-api",
		SourceRef:      "csapi://commands/route-42",
	}

	survey := Intent{
		NativeID:       "csapi-command-survey-7",
		TargetAssetID:  target,
		Name:           "Start survey pattern 7",
		Kind:           "mavlink.survey",
		Status:         StatusRequested,
		Description:    "federated search-sector tasking",
		DesiredState:   `{"command":"survey","pattern":"grid-7"}`,
		Authority:      "upstream.federated",
		Priority:       50,
		ExpiresAt:      start.Add(2 * time.Minute),
		CorrelationID:  "csapi:req-survey-7",
		IdempotencyKey: "idem-survey-7",
		RequestedBy:    "csapi:federated",
		ObservedAt:     start.Add(10 * time.Second),
		Source:         "cs-api",
		SourceRef:      "csapi://commands/survey-7",
	}

	stale := Intent{
		NativeID:       "csapi-command-stale-1",
		TargetAssetID:  target,
		Name:           "Expired stale goto",
		Kind:           "mavlink.goto",
		Status:         StatusRequested,
		Description:    "intentionally stale upstream command",
		DesiredState:   `{"command":"goto","lat":38.901,"lon":-77.045}`,
		Authority:      "upstream.federated",
		Priority:       40,
		ExpiresAt:      start.Add(30 * time.Second),
		CorrelationID:  "csapi:req-stale-1",
		IdempotencyKey: "idem-stale-1",
		RequestedBy:    "csapi:federated",
		ObservedAt:     start.Add(5 * time.Second),
		Source:         "cs-api",
		SourceRef:      "csapi://commands/stale-1",
	}

	return []LifecycleReplayRecord{
		intentReplayRecord("command://fixture/hadr-command/0001-route-requested", route),
		nativeStatusReplayRecord("command://fixture/hadr-command/0002-route-accepted", route.NativeID, "accepted", start.Add(3*time.Second), "command=400 progress=100 target=255/1"),
		nativeStatusReplayRecord("command://fixture/hadr-command/0003-route-executing", route.NativeID, "in_progress", start.Add(20*time.Second), "command=400 progress=50 target=255/1"),
		{
			Ref:    "command://fixture/hadr-command/0004-route-cancel-requested",
			Source: "command:fixture:hadr-command",
			Kind:   ReplayCancellation,
			Cancellation: &CancellationRequest{
				NativeID:       route.NativeID,
				TargetAssetID:  target,
				Authority:      "local.operator",
				Priority:       95,
				ExpiresAt:      start.Add(3 * time.Minute),
				CorrelationID:  "ui:cancel-route-42",
				IdempotencyKey: "cancel-idem-route-42",
				RequestedBy:    "operator:lead",
				Reason:         "airspace conflict",
				ObservedAt:     start.Add(60 * time.Second),
				Source:         "local-ui",
				SourceRef:      "ui://commands/cancel/route-42",
			},
		},
		nativeStatusReplayRecord("command://fixture/hadr-command/0005-route-cancelled", route.NativeID, "cancelled", start.Add(70*time.Second), "command=400 result=cancelled target=255/1"),
		intentReplayRecord("command://fixture/hadr-command/0006-survey-requested", survey),
		nativeStatusReplayRecord("command://fixture/hadr-command/0007-survey-accepted", survey.NativeID, "accepted", start.Add(12*time.Second), "mission accepted target=255/1"),
		{
			Ref:    "command://fixture/hadr-command/0008-survey-timeout",
			Source: "command:fixture:hadr-command",
			Kind:   ReplayDeadline,
			Deadline: &DeadlineEvidence{
				NativeID:   survey.NativeID,
				ObservedAt: survey.ExpiresAt.Add(10 * time.Second),
				Reason:     "native status window elapsed",
				Source:     "command.deadline.fixture",
				SourceRef:  "timer://command/survey-7",
			},
		},
		intentReplayRecord("command://fixture/hadr-command/0009-stale-requested", stale),
		{
			Ref:    "command://fixture/hadr-command/0010-stale-expired",
			Source: "command:fixture:hadr-command",
			Kind:   ReplayDeadline,
			Deadline: &DeadlineEvidence{
				NativeID:   stale.NativeID,
				ObservedAt: stale.ExpiresAt.Add(time.Second),
				Reason:     "no acceptance before ttl",
				Source:     "command.deadline.fixture",
				SourceRef:  "timer://command/stale-1",
			},
		},
	}
}

func MarshalLifecycleReplay(records []LifecycleReplayRecord) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	for index, record := range records {
		if err := record.validate(); err != nil {
			return nil, fmt.Errorf("validate lifecycle replay record %d: %w", index+1, err)
		}
		if err := encoder.Encode(record); err != nil {
			return nil, fmt.Errorf("encode lifecycle replay record %q: %w", record.Ref, err)
		}
	}
	return buf.Bytes(), nil
}

func LoadLifecycleReplay(path string) ([]LifecycleReplayRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open command lifecycle replay: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	records := make([]LifecycleReplayRecord, 0)
	for {
		var record LifecycleReplayRecord
		if err := decoder.Decode(&record); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode lifecycle replay record %d: %w", len(records)+1, err)
		}
		if err := record.validate(); err != nil {
			return nil, fmt.Errorf("decode lifecycle replay record %d: %w", len(records)+1, err)
		}
		records = append(records, record)
	}
	return records, nil
}

func intentReplayRecord(ref string, intent Intent) LifecycleReplayRecord {
	return LifecycleReplayRecord{
		Ref:    ref,
		Source: "command:fixture:hadr-command",
		Kind:   ReplayIntent,
		Intent: &intent,
	}
}

func nativeStatusReplayRecord(ref string, nativeID string, status string, observedAt time.Time, detail string) LifecycleReplayRecord {
	return LifecycleReplayRecord{
		Ref:    ref,
		Source: "command:fixture:hadr-command",
		Kind:   ReplayNativeStatus,
		NativeStatus: &NativeStatusEvidence{
			NativeID:     nativeID,
			Protocol:     "mavlink",
			NativeStatus: status,
			Detail:       detail,
			ObservedAt:   observedAt,
			Source:       "mavlink.command_ack",
			SourceRef:    "mavlink://fixture/hadr-command/" + entityToken(nativeID) + "/" + entityToken(status),
		},
	}
}

func (r LifecycleReplayRecord) validate() error {
	if strings.TrimSpace(r.Ref) == "" {
		return fmt.Errorf("missing ref")
	}
	if strings.TrimSpace(r.Source) == "" {
		return fmt.Errorf("record %q missing source", r.Ref)
	}
	if r.Kind == "" {
		return fmt.Errorf("record %q missing kind", r.Ref)
	}
	if countReplayPayloads(r) != 1 {
		return fmt.Errorf("record %q must have exactly one payload", r.Ref)
	}

	switch r.Kind {
	case ReplayIntent:
		if r.Intent == nil {
			return fmt.Errorf("record %q missing intent payload", r.Ref)
		}
		if err := r.Intent.validate(); err != nil {
			return fmt.Errorf("intent payload: %w", err)
		}
	case ReplayNativeStatus:
		if r.NativeStatus == nil {
			return fmt.Errorf("record %q missing native status payload", r.Ref)
		}
		return validateReplayNativeStatus(*r.NativeStatus)
	case ReplayCancellation:
		if r.Cancellation == nil {
			return fmt.Errorf("record %q missing cancellation payload", r.Ref)
		}
		return validateReplayCancellation(*r.Cancellation)
	case ReplayDeadline:
		if r.Deadline == nil {
			return fmt.Errorf("record %q missing deadline payload", r.Ref)
		}
		return validateReplayDeadline(*r.Deadline)
	default:
		return fmt.Errorf("record %q has unsupported kind %q", r.Ref, r.Kind)
	}
	return nil
}

func countReplayPayloads(r LifecycleReplayRecord) int {
	var count int
	if r.Intent != nil {
		count++
	}
	if r.NativeStatus != nil {
		count++
	}
	if r.Cancellation != nil {
		count++
	}
	if r.Deadline != nil {
		count++
	}
	return count
}

func validateReplayNativeStatus(evidence NativeStatusEvidence) error {
	if strings.TrimSpace(evidence.NativeID) == "" {
		return fmt.Errorf("native status native_id is required")
	}
	if strings.TrimSpace(evidence.Protocol) == "" {
		return fmt.Errorf("native status protocol is required")
	}
	if strings.TrimSpace(evidence.NativeStatus) == "" {
		return fmt.Errorf("native status status is required")
	}
	if evidence.ObservedAt.IsZero() {
		return fmt.Errorf("native status observed_at is required")
	}
	return nil
}

func validateReplayCancellation(request CancellationRequest) error {
	if strings.TrimSpace(request.NativeID) == "" {
		return fmt.Errorf("cancellation native_id is required")
	}
	if strings.TrimSpace(request.TargetAssetID) == "" {
		return fmt.Errorf("cancellation target asset id is required")
	}
	if strings.TrimSpace(request.Authority) == "" {
		return fmt.Errorf("cancellation authority is required")
	}
	if request.Priority < 1 || request.Priority > 100 {
		return fmt.Errorf("cancellation priority must be between 1 and 100")
	}
	if request.ExpiresAt.IsZero() {
		return fmt.Errorf("cancellation expires_at is required")
	}
	if request.ObservedAt.IsZero() {
		return fmt.Errorf("cancellation observed_at is required")
	}
	if !request.ExpiresAt.After(request.ObservedAt.UTC()) {
		return fmt.Errorf("cancellation expires_at must be after observed_at")
	}
	if strings.TrimSpace(request.CorrelationID) == "" {
		return fmt.Errorf("cancellation correlation_id is required")
	}
	if strings.TrimSpace(request.IdempotencyKey) == "" {
		return fmt.Errorf("cancellation idempotency_key is required")
	}
	if strings.TrimSpace(request.RequestedBy) == "" {
		return fmt.Errorf("cancellation requested_by is required")
	}
	return nil
}

func validateReplayDeadline(evidence DeadlineEvidence) error {
	if strings.TrimSpace(evidence.NativeID) == "" {
		return fmt.Errorf("deadline native_id is required")
	}
	if evidence.ObservedAt.IsZero() {
		return fmt.Errorf("deadline observed_at is required")
	}
	return nil
}
