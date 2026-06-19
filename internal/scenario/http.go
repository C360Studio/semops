package scenario

import (
	"encoding/json"
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
	return mux
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
