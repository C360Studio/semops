//go:build ignore
// +build ignore

package vocabulary

// Resolution strategies for entity deduplication and conflict resolution.
// These constants define how the system should handle conflicting entity data
// when multiple sources provide information about the same entity.
const (
	// ResolutionStrategyLatestTimestamp prioritizes the most recent data
	// based on timestamp comparison. Suitable for frequently updated data
	// where temporal ordering matters.
	ResolutionStrategyLatestTimestamp = "latest_timestamp"

	// ResolutionStrategyHighestConfidence prioritizes data with the highest
	// confidence score. Suitable when data quality varies significantly
	// between sources.
	ResolutionStrategyHighestConfidence = "highest_confidence"

	// ResolutionStrategyManualReview requires human intervention to resolve
	// conflicts. Used when automated resolution might introduce errors or
	// when business logic is complex.
	ResolutionStrategyManualReview = "manual_review"
)
