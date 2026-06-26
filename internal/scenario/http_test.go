package scenario

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusHandlerHealthCodes(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   State
		want int
	}{
		{name: "idle", in: StateIdle, want: http.StatusServiceUnavailable},
		{name: "running", in: StateRunning, want: http.StatusServiceUnavailable},
		{name: "succeeded", in: StateSucceeded, want: http.StatusOK},
		{name: "failed", in: StateFailed, want: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewStatusHandler(func() Status {
				return Status{ScenarioID: "scenario-1", State: tc.in}
			})
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("status code = %d, want %d; body=%s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestStatusHandlerReturnsScenarioStatus(t *testing.T) {
	handler := NewStatusHandler(func() Status {
		return Status{
			ScenarioID:     "phase-1-hadr-flood-evacuation",
			State:          StateSucceeded,
			IngressMode:    IngressModeDirectGraphContract,
			CompletedSteps: 10,
			Summary: Summary{
				MAVLinkFrames:                 2,
				CoTEvents:                     4,
				CAPAlerts:                     4,
				ContractGraphMutationAttempts: 18,
				Mutations:                     18,
			},
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/scenario/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	var status Status
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.ScenarioID != "phase-1-hadr-flood-evacuation" ||
		status.State != StateSucceeded ||
		status.IngressMode != IngressModeDirectGraphContract ||
		status.CompletedSteps != 10 ||
		status.Summary.ContractGraphMutationAttempts != 18 ||
		status.Summary.Mutations != 18 {
		t.Fatalf("status = %+v", status)
	}
}
