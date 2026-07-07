package cop

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	SemLinkReadbackAuthorityScope = "semlink.readback.intent"
	SemLinkReadbackOperatorRole   = AuthenticatedAssociationReviewerRole
)

type SemLinkReadbackAuthorizer func(*http.Request) (SemLinkReadbackCaller, error)

type SemLinkReadbackCaller struct {
	ID              string
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
	return SemLinkReadbackCaller{
		ID:              id,
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
