//go:build ignore
// +build ignore

package vocabulary

// Relationship constants commonly used by robotics domain.
// These values match the constants defined in pkg/graph to maintain compatibility,
// but are defined independently to avoid import cycles.
const (
	// Spatial relationships - describe physical or logical location relationships
	RelTypeLocatedAt = "LOCATED_AT" // Entity is at a specific location
	RelTypeNear      = "NEAR"       // Entity is near another entity (proximity)

	// Operational relationships - describe functional relationships
	RelTypeExecuting    = "EXECUTING"     // Entity is executing/performing another entity
	RelTypeControlledBy = "CONTROLLED_BY" // Entity is controlled by another entity
	RelTypeMemberOf     = "MEMBER_OF"     // Entity is a member of another entity (groups, formations)

	// Component relationships - describe structural relationships
	RelTypePoweredBy = "POWERED_BY" // Entity is powered by another entity

	// Status relationships - describe state and reporting relationships
	RelTypeHasStatus = "HAS_STATUS" // Entity has a particular status
)

// Additional robotics-specific relationships not covered by the global graph vocabulary
const (
	// Communication relationships specific to robotics domain
	RelTypeCommunicatesWith = "COMMUNICATES_WITH" // Entity has communication link with another entity
)
