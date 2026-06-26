package cop

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	OperatorIDHeader             = "X-SemOps-Operator-ID"
	OperatorRoleHeader           = "X-SemOps-Operator-Role"
	OperatorAuthorityScopeHeader = "X-SemOps-Authority-Scope"

	OperatorIdentitySourceDefault = "local-default"
	OperatorIdentitySourceBody    = "request-body"
	OperatorIdentitySourceHeader  = "request-header"

	MaxOperatorIdentityLen = 128
)

type OperatorIdentity struct {
	ID             string
	ReviewerRole   string
	AuthorityScope string
	Authenticated  bool
	Source         string
}

type OperatorIdentityResolver func(*http.Request, string) (OperatorIdentity, error)

func ResolveLocalOperatorIdentity(r *http.Request, requestedID string) (OperatorIdentity, error) {
	id := strings.TrimSpace(requestedID)
	source := OperatorIdentitySourceBody
	if r != nil {
		if headerID := strings.TrimSpace(r.Header.Get(OperatorIDHeader)); headerID != "" {
			id = headerID
			source = OperatorIdentitySourceHeader
		}
		if role := strings.TrimSpace(r.Header.Get(OperatorRoleHeader)); role != "" && role != DefaultAssociationReviewerRole {
			return OperatorIdentity{}, fmt.Errorf(
				"operator role must be %q for MVP display-only reviews",
				DefaultAssociationReviewerRole,
			)
		}
		if scope := strings.TrimSpace(r.Header.Get(OperatorAuthorityScopeHeader)); scope != "" && scope != DefaultAssociationReviewAuthorityScope {
			return OperatorIdentity{}, fmt.Errorf(
				"operator authority_scope must be %q for MVP display-only reviews",
				DefaultAssociationReviewAuthorityScope,
			)
		}
	}
	if id == "" {
		id = DefaultAssociationReviewer
		source = OperatorIdentitySourceDefault
	}
	if err := validateOperatorIdentityID(id); err != nil {
		return OperatorIdentity{}, err
	}
	return OperatorIdentity{
		ID:             id,
		ReviewerRole:   DefaultAssociationReviewerRole,
		AuthorityScope: DefaultAssociationReviewAuthorityScope,
		Authenticated:  false,
		Source:         source,
	}, nil
}

func validateOperatorIdentityID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("operator identity is required")
	}
	if len(id) > MaxOperatorIdentityLen {
		return fmt.Errorf("operator identity exceeds %d characters", MaxOperatorIdentityLen)
	}
	if strings.ContainsFunc(id, func(r rune) bool {
		return r < ' ' || r == 0x7f
	}) {
		return fmt.Errorf("operator identity contains control characters")
	}
	return nil
}
