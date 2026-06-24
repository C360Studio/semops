package cop

import (
	"context"
	"errors"
	"testing"
	"time"

	fusionprojector "github.com/c360studio/semops/internal/projectors/fusion"
	copmodel "github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
)

func TestGraphAssociationReviewStoreWritesAuditBeforeLocalOverlay(t *testing.T) {
	writer := &recordingAssociationReviewWriter{}
	store, err := NewGraphAssociationReviewStore(NewMemoryAssociationReviewStore(), AssociationReviewGraphStoreConfig{
		Org:       "c360",
		Platform:  "edge",
		Projector: fusionprojector.NewProjector(fusionprojector.Config{}),
		Writer:    writer,
	})
	if err != nil {
		t.Fatalf("new graph review store: %v", err)
	}
	review := sampleAssociationReview(time.Date(2026, 6, 24, 2, 0, 0, 0, time.UTC))

	got, err := store.PutAssociationReview(context.Background(), review)
	if err != nil {
		t.Fatalf("put association review: %v", err)
	}
	if got.Decision != AssociationReviewAcknowledged ||
		got.AuthorityScope != DefaultAssociationReviewAuthorityScope {
		t.Fatalf("review = %+v", got)
	}
	if len(writer.plans) != 1 || len(writer.plans[0].Mutations) != 1 {
		t.Fatalf("plans = %+v", writer.plans)
	}
	create := writer.plans[0].Mutations[0].Create
	if create.Entity == nil || create.Entity.MessageType.String() != copmodel.FusionAssociationReviewContract().MessageType {
		t.Fatalf("create = %+v", create)
	}
	reviews, err := store.ListAssociationReviews(context.Background())
	if err != nil {
		t.Fatalf("list reviews: %v", err)
	}
	if len(reviews) != 1 || reviews[0].AssociationID != review.AssociationID {
		t.Fatalf("local reviews = %+v", reviews)
	}
}

func TestGraphAssociationReviewStoreReconcilesExistingReviewBirth(t *testing.T) {
	review := sampleAssociationReview(time.Date(2026, 6, 24, 2, 5, 0, 0, time.UTC))
	entityID := fusionprojector.AssociationReviewEntityID("c360", "edge", review.AssociationID)
	writer := &recordingAssociationReviewWriter{
		errs: []error{&fusionprojector.MutationFailureError{
			Operation: "create_with_triples",
			Kind:      fusionprojector.MutationCreate,
			EntityID:  entityID,
			ErrorCode: graph.ErrorCodeEntityExists,
			Message:   "entity exists",
		}},
	}
	store, err := NewGraphAssociationReviewStore(NewMemoryAssociationReviewStore(), AssociationReviewGraphStoreConfig{
		Org:       "c360",
		Platform:  "edge",
		Projector: fusionprojector.NewProjector(fusionprojector.Config{}),
		Writer:    writer,
	})
	if err != nil {
		t.Fatalf("new graph review store: %v", err)
	}

	if _, err := store.PutAssociationReview(context.Background(), review); err != nil {
		t.Fatalf("put association review: %v", err)
	}
	if len(writer.plans) != 2 {
		t.Fatalf("plans = %d, want create then update", len(writer.plans))
	}
	if writer.plans[0].Mutations[0].Kind != fusionprojector.MutationCreate {
		t.Fatalf("first mutation = %q, want create", writer.plans[0].Mutations[0].Kind)
	}
	if writer.plans[1].Mutations[0].Kind != fusionprojector.MutationUpdate {
		t.Fatalf("second mutation = %q, want update", writer.plans[1].Mutations[0].Kind)
	}
}

func TestGraphAssociationReviewStoreSkipsLocalOverlayWhenAuditWriteFails(t *testing.T) {
	writer := &recordingAssociationReviewWriter{errs: []error{errors.New("graph down")}}
	store, err := NewGraphAssociationReviewStore(NewMemoryAssociationReviewStore(), AssociationReviewGraphStoreConfig{
		Org:       "c360",
		Platform:  "edge",
		Projector: fusionprojector.NewProjector(fusionprojector.Config{}),
		Writer:    writer,
	})
	if err != nil {
		t.Fatalf("new graph review store: %v", err)
	}

	if _, err := store.PutAssociationReview(context.Background(), sampleAssociationReview(time.Now().UTC())); err == nil {
		t.Fatal("expected graph write failure")
	}
	reviews, err := store.ListAssociationReviews(context.Background())
	if err != nil {
		t.Fatalf("list reviews: %v", err)
	}
	if len(reviews) != 0 {
		t.Fatalf("local reviews = %+v, want none", reviews)
	}
}

type recordingAssociationReviewWriter struct {
	plans []fusionprojector.Plan
	errs  []error
}

func (w *recordingAssociationReviewWriter) Apply(_ context.Context, plan fusionprojector.Plan) error {
	w.plans = append(w.plans, plan)
	if len(w.errs) == 0 {
		return nil
	}
	err := w.errs[0]
	w.errs = w.errs[1:]
	return err
}

func sampleAssociationReview(reviewedAt time.Time) AssociationReview {
	return AssociationReview{
		AssociationID:  "c360.edge.cop.fusion.association.mavlink-to-tak",
		Decision:       AssociationReviewAcknowledged,
		ReviewedBy:     "operator:lead",
		ReviewedAt:     reviewedAt,
		ReviewerRole:   DefaultAssociationReviewerRole,
		AuthorityScope: DefaultAssociationReviewAuthorityScope,
		ConflictPolicy: DefaultAssociationReviewConflictPolicy,
		Comment:        "reviewed in COP",
	}
}
