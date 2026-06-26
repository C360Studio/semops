package scenario

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestStatusHandlerReturnsFailClosedScenarioControls(t *testing.T) {
	handler := NewStatusHandler(func() Status {
		return Status{
			ScenarioID: "phase-1-hadr-flood-evacuation",
			State:      StateSucceeded,
			Checkpoints: []CheckpointEvaluation{{
				ID:         "operator-review",
				ClaimScope: ClaimOperatorControl,
				State:      CheckpointDeclared,
			}},
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/scenario/controls", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	var controls ControlCapabilities
	if err := json.NewDecoder(rec.Body).Decode(&controls); err != nil {
		t.Fatalf("decode controls: %v", err)
	}
	if controls.Enabled ||
		controls.State != "blocked" ||
		controls.RequiredClaimScope != ClaimOperatorControl ||
		controls.CheckpointID != "operator-review" ||
		controls.CheckpointState != string(CheckpointDeclared) ||
		len(controls.SupportedActions) != 4 ||
		!strings.Contains(controls.Reason, "reviewed operator_scenario_control checkpoint") {
		t.Fatalf("controls = %+v", controls)
	}
}

func TestStatusHandlerRejectsScenarioControlActions(t *testing.T) {
	handler := NewStatusHandler(func() Status {
		return Status{ScenarioID: "phase-1-hadr-flood-evacuation", State: StateSucceeded}
	})
	req := httptest.NewRequest(
		http.MethodPost,
		"/scenario/controls",
		bytes.NewBufferString(`{"action":"reset"}`),
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status code = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
	var result ControlResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Accepted ||
		result.Action != ControlActionReset ||
		result.State != "blocked" ||
		result.Controls.Enabled ||
		result.Controls.RequiredClaimScope != ClaimOperatorControl {
		t.Fatalf("result = %+v", result)
	}
}

func TestStatusHandlerRejectsUnsupportedScenarioControlActions(t *testing.T) {
	handler := NewStatusHandler(func() Status { return Status{ScenarioID: "scenario-1"} })
	req := httptest.NewRequest(
		http.MethodPost,
		"/scenario/controls",
		bytes.NewBufferString(`{"action":"launch"}`),
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "unsupported scenario control action") {
		t.Fatalf("response = %d %s", rec.Code, rec.Body.String())
	}
}
