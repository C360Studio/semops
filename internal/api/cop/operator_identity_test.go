package cop

import (
	"net/http"
	"strings"
	"testing"
)

func TestResolveLocalOperatorIdentityUsesDefaultWithoutSetup(t *testing.T) {
	identity, err := ResolveLocalOperatorIdentity(nil, "")
	if err != nil {
		t.Fatalf("resolve identity: %v", err)
	}
	if identity.ID != DefaultAssociationReviewer ||
		identity.ReviewerRole != DefaultAssociationReviewerRole ||
		identity.AuthorityScope != DefaultAssociationReviewAuthorityScope ||
		identity.Authenticated ||
		identity.Source != OperatorIdentitySourceDefault {
		t.Fatalf("identity = %+v", identity)
	}
}

func TestResolveLocalOperatorIdentityUsesHeaderID(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/cop/associations/a/review", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set(OperatorIDHeader, "operator:incident-command")
	req.Header.Set(OperatorRoleHeader, DefaultAssociationReviewerRole)
	req.Header.Set(OperatorAuthorityScopeHeader, DefaultAssociationReviewAuthorityScope)

	identity, err := ResolveLocalOperatorIdentity(req, "operator:body")
	if err != nil {
		t.Fatalf("resolve identity: %v", err)
	}
	if identity.ID != "operator:incident-command" ||
		identity.Source != OperatorIdentitySourceHeader ||
		identity.Authenticated {
		t.Fatalf("identity = %+v", identity)
	}
}

func TestResolveLocalOperatorIdentityRejectsAuthorityEscalation(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/cop/associations/a/review", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set(OperatorAuthorityScopeHeader, "command.execute")

	_, err = ResolveLocalOperatorIdentity(req, "operator:lead")
	if err == nil || !strings.Contains(err.Error(), "display-only reviews") {
		t.Fatalf("error = %v, want display-only rejection", err)
	}
}
