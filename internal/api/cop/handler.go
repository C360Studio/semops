package cop

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type SnapshotProvider interface {
	Snapshot(context.Context) (Snapshot, error)
}

type Handler struct {
	provider                 SnapshotProvider
	runtimeProvider          RuntimeProvider
	reviewStore              AssociationReviewStore
	operatorIdentityResolver OperatorIdentityResolver
	now                      func() time.Time
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

func WithAssociationReviewStore(store AssociationReviewStore) Option {
	return func(h *Handler) {
		h.reviewStore = store
	}
}

func WithOperatorIdentityResolver(resolver OperatorIdentityResolver) Option {
	return func(h *Handler) {
		h.operatorIdentityResolver = resolver
	}
}

func NewHandler(provider SnapshotProvider, opts ...Option) (*Handler, error) {
	if provider == nil {
		return nil, fmt.Errorf("cop api requires a snapshot provider")
	}
	handler := &Handler{
		provider:                 provider,
		reviewStore:              NewMemoryAssociationReviewStore(),
		operatorIdentityResolver: ResolveLocalOperatorIdentity,
		now:                      func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		if opt != nil {
			opt(handler)
		}
	}
	if handler.reviewStore == nil {
		handler.reviewStore = NewMemoryAssociationReviewStore()
	}
	if handler.operatorIdentityResolver == nil {
		handler.operatorIdentityResolver = ResolveLocalOperatorIdentity
	}
	return handler, nil
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /api/cop/snapshot", h.snapshot)
	mux.HandleFunc("GET /api/cop/runtime", h.runtime)
	mux.HandleFunc("POST /api/cop/associations/{associationID}/review", h.reviewAssociation)
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
	if err := h.applyAssociationReviews(r.Context(), &snapshot); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (h *Handler) runtime(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, BuildRuntimeSnapshot(h.now(), h.runtimeProvider))
}

func (h *Handler) reviewAssociation(w http.ResponseWriter, r *http.Request) {
	associationID := strings.TrimSpace(r.PathValue("associationID"))
	if associationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "association id is required"})
		return
	}
	var request associationReviewRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid association review request"})
		return
	}
	decision := normalizeAssociationReviewDecision(request.Decision)
	switch decision {
	case AssociationReviewAcknowledged, AssociationReviewChallenged:
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf(
				"decision must be %q or %q",
				AssociationReviewAcknowledged,
				AssociationReviewChallenged,
			),
		})
		return
	}
	snapshot, err := h.provider.Snapshot(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !snapshotHasAssociation(snapshot, associationID) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "association not found"})
		return
	}
	identity, err := h.operatorIdentityResolver(r, request.ReviewedBy)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	review := AssociationReview{
		AssociationID:  associationID,
		Decision:       decision,
		ReviewedBy:     identity.ID,
		ReviewedAt:     h.now().UTC(),
		ReviewerRole:   identity.ReviewerRole,
		AuthorityScope: identity.AuthorityScope,
		ConflictPolicy: DefaultAssociationReviewConflictPolicy,
		Comment:        strings.TrimSpace(request.Comment),
	}
	review, err = h.reviewStore.PutAssociationReview(r.Context(), review)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, review)
}

type associationReviewRequest struct {
	Decision   string `json:"decision"`
	ReviewedBy string `json:"reviewed_by,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

func (h *Handler) applyAssociationReviews(ctx context.Context, snapshot *Snapshot) error {
	if h == nil || h.reviewStore == nil || snapshot == nil || len(snapshot.Associations) == 0 {
		return nil
	}
	reviews, err := h.reviewStore.ListAssociationReviews(ctx)
	if err != nil {
		return fmt.Errorf("list association reviews: %w", err)
	}
	reviewsByAssociation := make(map[string]AssociationReview, len(reviews))
	for _, review := range reviews {
		reviewsByAssociation[review.AssociationID] = review
	}
	for index := range snapshot.Associations {
		review, ok := reviewsByAssociation[snapshot.Associations[index].ID]
		if !ok {
			continue
		}
		snapshot.Associations[index].OperatorReview = &review
	}
	return nil
}

func snapshotHasAssociation(snapshot Snapshot, associationID string) bool {
	for _, association := range snapshot.Associations {
		if association.ID == associationID {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
