package scenario

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type StatusFunc func() Status

func NewStatusHandler(status StatusFunc) http.Handler {
	if status == nil {
		status = func() Status {
			return Status{State: StateFailed, LastError: "scenario status provider is not configured"}
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		current := status()
		switch current.State {
		case StateSucceeded:
			writeJSON(w, http.StatusOK, current)
		case StateFailed:
			writeJSON(w, http.StatusInternalServerError, current)
		default:
			writeJSON(w, http.StatusServiceUnavailable, current)
		}
	})
	mux.HandleFunc("/scenario/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, status())
	})
	mux.HandleFunc("/scenario/controls", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, EvaluateControlCapabilities(status()))
		case http.MethodPost:
			var request ControlRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("decode control request: %v", err)})
				return
			}
			if !ValidControlAction(request.Action) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("unsupported scenario control action %q", request.Action)})
				return
			}
			writeJSON(w, http.StatusConflict, RejectControlRequest(status(), request.Action))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	return mux
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
