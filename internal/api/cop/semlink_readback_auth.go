package cop

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	SemLinkReadbackAuthorityScope = "semlink.readback.intent"
	SemLinkReadbackOperatorRole   = AuthenticatedAssociationReviewerRole
	SemLinkCompanionNodeIDHeader  = "X-SemOps-SemLink-Companion-Node-ID"
	SemLinkMeshNodeIDHeader       = "X-SemOps-SemLink-Mesh-Node-ID"
)

type SemLinkReadbackAuthorizer func(*http.Request) (SemLinkReadbackCaller, error)

type SemLinkReadbackCaller struct {
	ID              string
	CompanionNodeID string
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
	companionNodeID := firstSemLinkNodeHeader(r)
	if err := validateSemLinkCompanionNodeID(companionNodeID); err != nil {
		return SemLinkReadbackCaller{}, semLinkReadbackAuthError(http.StatusForbidden, err.Error())
	}
	return SemLinkReadbackCaller{
		ID:              id,
		CompanionNodeID: companionNodeID,
		MeshNodeID:      companionNodeID,
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

func firstSemLinkNodeHeader(r *http.Request) string {
	if r == nil {
		return ""
	}
	if value := strings.TrimSpace(r.Header.Get(SemLinkCompanionNodeIDHeader)); value != "" {
		return value
	}
	return strings.TrimSpace(r.Header.Get(SemLinkMeshNodeIDHeader))
}

func validateSemLinkCompanionNodeID(companionNodeID string) error {
	if strings.TrimSpace(companionNodeID) == "" {
		return fmt.Errorf("%s is required for SemLink readback", SemLinkCompanionNodeIDHeader)
	}
	if len(companionNodeID) > MaxOperatorIdentityLen {
		return fmt.Errorf("%s exceeds %d characters", SemLinkCompanionNodeIDHeader, MaxOperatorIdentityLen)
	}
	if strings.ContainsFunc(companionNodeID, func(r rune) bool {
		return r < ' ' || r == 0x7f
	}) {
		return fmt.Errorf("%s contains control characters", SemLinkCompanionNodeIDHeader)
	}
	return nil
}
