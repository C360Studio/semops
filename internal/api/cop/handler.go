package cop

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SnapshotProvider interface {
	Snapshot(context.Context) (Snapshot, error)
}

type Handler struct {
	provider        SnapshotProvider
	runtimeProvider RuntimeProvider
	now             func() time.Time
}

type Option func(*Handler)

func WithRuntimeProvider(provider RuntimeProvider) Option {
	return func(h *Handler) {
		h.runtimeProvider = provider
	}
}

func WithClock(now func() time.Time) Option {
	return func(h *Handler) {
		if now != nil {
			h.now = now
		}
	}
}

func NewHandler(provider SnapshotProvider, opts ...Option) (*Handler, error) {
	if provider == nil {
		return nil, fmt.Errorf("cop api requires a snapshot provider")
	}
	handler := &Handler{
		provider: provider,
		now:      func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		if opt != nil {
			opt(handler)
		}
	}
	return handler, nil
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /api/cop/snapshot", h.snapshot)
	mux.HandleFunc("GET /api/cop/runtime", h.runtime)
	return mux
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) snapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.provider.Snapshot(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	if snapshot.GeneratedAt.IsZero() {
		snapshot.GeneratedAt = time.Now().UTC()
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (h *Handler) runtime(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, BuildRuntimeSnapshot(h.now(), h.runtimeProvider))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
