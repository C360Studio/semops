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
		identity.AuthorityDomain != DefaultAssociationReviewAuthorityDomain ||
		identity.ConflictPolicy != DefaultAssociationReviewConflictPolicy ||
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
		identity.AuthorityDomain != DefaultAssociationReviewAuthorityDomain ||
		identity.Authenticated {
		t.Fatalf("identity = %+v", identity)
	}
}

func TestResolveTrustedHeaderOperatorIdentity(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/cop/associations/a/review", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set(OperatorAuthenticatedHeader, "true")
	req.Header.Set(OperatorIDHeader, "operator:incident-command")
	req.Header.Set(OperatorRoleHeader, AuthenticatedAssociationReviewerRole)
	req.Header.Set(OperatorAuthorityScopeHeader, AuthenticatedAssociationReviewScope)
	req.Header.Set(OperatorAuthorityDomainHeader, "incident-command")

	identity, err := ResolveTrustedHeaderOperatorIdentity(req, "operator:body")
	if err != nil {
		t.Fatalf("resolve identity: %v", err)
	}
	if identity.ID != "operator:incident-command" ||
		identity.ReviewerRole != AuthenticatedAssociationReviewerRole ||
		identity.AuthorityScope != AuthenticatedAssociationReviewScope ||
		identity.AuthorityDomain != "incident-command" ||
		identity.ConflictPolicy != AuthenticatedAssociationReviewPolicy ||
		!identity.Authenticated ||
		identity.Source != OperatorIdentitySourceTrusted {
		t.Fatalf("identity = %+v", identity)
	}
}

func TestResolveTrustedHeaderOperatorIdentityRejectsMissingAuthentication(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/cop/associations/a/review", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set(OperatorIDHeader, "operator:incident-command")
	req.Header.Set(OperatorAuthorityDomainHeader, "incident-command")

	_, err = ResolveTrustedHeaderOperatorIdentity(req, "")
	if err == nil || !strings.Contains(err.Error(), OperatorAuthenticatedHeader) {
		t.Fatalf("error = %v, want authenticated-header rejection", err)
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
