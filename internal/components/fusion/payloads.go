package fusion

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	fusionassociation "github.com/c360studio/semops/internal/fusion/association"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

const DefaultCandidateSubject = "semops.fusion.track_candidates"

var CandidateBatchType = message.Type{
	Domain:   "semops",
	Category: "fusion_track_candidates",
	Version:  "v1",
}

type CandidateBatchPayload struct {
	Source     string                               `json:"source"`
	BatchID    string                               `json:"batch_id,omitempty"`
	ObservedAt time.Time                            `json:"observed_at"`
	Primary    []fusionassociation.TrackObservation `json:"primary"`
	Candidates []fusionassociation.TrackObservation `json:"candidates"`
}

func NewCandidateBatchPayload(
	source string,
	batchID string,
	observedAt time.Time,
	primary []fusionassociation.TrackObservation,
	candidates []fusionassociation.TrackObservation,
) *CandidateBatchPayload {
	return &CandidateBatchPayload{
		Source:     source,
		BatchID:    batchID,
		ObservedAt: observedAt.UTC(),
		Primary:    cloneTrackObservations(primary),
		Candidates: cloneTrackObservations(candidates),
	}
}

func (p *CandidateBatchPayload) Schema() message.Type {
	return CandidateBatchType
}

func (p *CandidateBatchPayload) Validate() error {
	if p == nil {
		return errors.New("fusion candidate batch payload is nil")
	}
	if p.Source == "" {
		return errors.New("fusion candidate batch source is required")
	}
	if p.ObservedAt.IsZero() {
		return errors.New("fusion candidate batch observed_at is required")
	}
	if len(p.Primary) == 0 {
		return errors.New("fusion candidate batch primary observations are required")
	}
	if len(p.Candidates) == 0 {
		return errors.New("fusion candidate batch candidate observations are required")
	}
	if err := validateTrackObservations("primary", p.Primary); err != nil {
		return err
	}
	return validateTrackObservations("candidates", p.Candidates)
}

func (p *CandidateBatchPayload) TrackObservationsCopy() ([]fusionassociation.TrackObservation, []fusionassociation.TrackObservation, error) {
	if err := p.Validate(); err != nil {
		return nil, nil, err
	}
	return cloneTrackObservations(p.Primary), cloneTrackObservations(p.Candidates), nil
}

func (p *CandidateBatchPayload) MarshalJSON() ([]byte, error) {
	type alias CandidateBatchPayload
	return json.Marshal((*alias)(p))
}

func (p *CandidateBatchPayload) UnmarshalJSON(data []byte) error {
	type alias CandidateBatchPayload
	return json.Unmarshal(data, (*alias)(p))
}

func RegisterPayloads(registry *payloadregistry.Registry) error {
	if registry == nil {
		return errors.New("payload registry is nil")
	}
	if _, ok := registry.GetRegistration(CandidateBatchType.Key()); ok {
		return nil
	}
	return registry.Register(&payloadregistry.Registration{
		Factory:     func() any { return &CandidateBatchPayload{} },
		Domain:      CandidateBatchType.Domain,
		Category:    CandidateBatchType.Category,
		Version:     CandidateBatchType.Version,
		Description: "Fusion association candidate batch emitted by a bounded COP track-candidate producer",
	})
}

func validateTrackObservations(role string, observations []fusionassociation.TrackObservation) error {
	for i, observation := range observations {
		switch {
		case observation.ID == "":
			return fmt.Errorf("fusion %s observation %d id is required", role, i)
		case observation.Source == "":
			return fmt.Errorf("fusion %s observation %d source is required", role, i)
		case observation.ObservedAt.IsZero():
			return fmt.Errorf("fusion %s observation %d observed_at is required", role, i)
		case observation.Position.Lat < -90 || observation.Position.Lat > 90:
			return fmt.Errorf("fusion %s observation %d latitude is invalid", role, i)
		case observation.Position.Lon < -180 || observation.Position.Lon > 180:
			return fmt.Errorf("fusion %s observation %d longitude is invalid", role, i)
		}
	}
	return nil
}

func cloneTrackObservations(in []fusionassociation.TrackObservation) []fusionassociation.TrackObservation {
	out := make([]fusionassociation.TrackObservation, len(in))
	copy(out, in)
	for i := range out {
		out[i].ObservedAt = out[i].ObservedAt.UTC()
	}
	return out
}
