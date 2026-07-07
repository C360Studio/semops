package cop

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	SemLinkReadbackAuthorityScope = "semlink.readback.intent"
	SemLinkReadbackOperatorRole   = AuthenticatedAssociationReviewerRole
	SemLinkMeshNodeIDHeader       = "X-SemOps-SemLink-Mesh-Node-ID"
)

type SemLinkReadbackAuthorizer func(*http.Request) (SemLinkReadbackCaller, error)

type SemLinkReadbackCaller struct {
	ID              string
	MeshNodeID      string
	AuthorityScope  string
	AuthorityDomain string
	Authenticated   bool
}

type SemLinkReadbackAuthError struct {
	Status  int
	Message string
}

func (e *SemLinkReadbackAuthError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func RequireTrustedSemLinkReadbackHeaders(r *http.Request) (SemLinkReadbackCaller, error) {
	if r == nil {
		return SemLinkReadbackCaller{}, semLinkReadbackAuthError(
			http.StatusUnauthorized,
			"semlink readback requires trusted request headers",
		)
	}
	if !trustedAuthenticatedHeader(r.Header.Get(OperatorAuthenticatedHeader)) {
		return SemLinkReadbackCaller{}, semLinkReadbackAuthError(
			http.StatusUnauthorized,
			fmt.Sprintf("%s must be true for SemLink readback", OperatorAuthenticatedHeader),
		)
	}
	id := strings.TrimSpace(r.Header.Get(OperatorIDHeader))
	if err := validateOperatorIdentityID(id); err != nil {
		return SemLinkReadbackCaller{}, semLinkReadbackAuthError(http.StatusUnauthorized, err.Error())
	}
	role := strings.TrimSpace(r.Header.Get(OperatorRoleHeader))
	if role != "" && role != SemLinkReadbackOperatorRole {
		return SemLinkReadbackCaller{}, semLinkReadbackAuthError(
			http.StatusForbidden,
			fmt.Sprintf("%s must be %q for SemLink readback", OperatorRoleHeader, SemLinkReadbackOperatorRole),
		)
	}
	scope := strings.TrimSpace(r.Header.Get(OperatorAuthorityScopeHeader))
	if scope != SemLinkReadbackAuthorityScope {
		return SemLinkReadbackCaller{}, semLinkReadbackAuthError(
			http.StatusForbidden,
			fmt.Sprintf("%s must be %q for SemLink readback", OperatorAuthorityScopeHeader, SemLinkReadbackAuthorityScope),
		)
	}
	domain := strings.TrimSpace(r.Header.Get(OperatorAuthorityDomainHeader))
	if err := validateAuthorityDomain(domain); err != nil {
		return SemLinkReadbackCaller{}, semLinkReadbackAuthError(http.StatusForbidden, err.Error())
	}
	meshNodeID := strings.TrimSpace(r.Header.Get(SemLinkMeshNodeIDHeader))
	if err := validateSemLinkMeshNodeID(meshNodeID); err != nil {
		return SemLinkReadbackCaller{}, semLinkReadbackAuthError(http.StatusForbidden, err.Error())
	}
	return SemLinkReadbackCaller{
		ID:              id,
		MeshNodeID:      meshNodeID,
		AuthorityScope:  scope,
		AuthorityDomain: domain,
		Authenticated:   true,
	}, nil
}

func semLinkReadbackAuthError(status int, message string) *SemLinkReadbackAuthError {
	if status == 0 {
		status = http.StatusUnauthorized
	}
	return &SemLinkReadbackAuthError{Status: status, Message: message}
}

func validateSemLinkMeshNodeID(meshNodeID string) error {
	if strings.TrimSpace(meshNodeID) == "" {
		return fmt.Errorf("%s is required for SemLink readback", SemLinkMeshNodeIDHeader)
	}
	if len(meshNodeID) > MaxOperatorIdentityLen {
		return fmt.Errorf("%s exceeds %d characters", SemLinkMeshNodeIDHeader, MaxOperatorIdentityLen)
	}
	if strings.ContainsFunc(meshNodeID, func(r rune) bool {
		return r < ' ' || r == 0x7f
	}) {
		return fmt.Errorf("%s contains control characters", SemLinkMeshNodeIDHeader)
	}
	return nil
}
