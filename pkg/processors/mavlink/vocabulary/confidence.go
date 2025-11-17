package vocabulary

// GPS confidence levels based on fix quality and accuracy.
// These constants provide standardized confidence values for positioning data
// ranging from no fix to high-precision RTK corrections.
const (
	GPSConfidenceNoFix  = 0.5  // No GPS fix available, low confidence
	GPSConfidence2DFix  = 0.7  // 2D GPS fix, moderate confidence
	GPSConfidence3DFix  = 0.95 // 3D GPS fix with good satellite coverage
	GPSConfidenceRTKFix = 0.99 // RTK-corrected GPS fix, highest confidence
)

// Battery confidence levels based on data source and measurement method.
// Direct measurements from battery management systems have higher confidence
// than inferred or estimated values.
const (
	BatteryConfidenceDirect   = 1.0 // Direct measurement from battery management system
	BatteryConfidenceInferred = 0.8 // Inferred from voltage/current measurements
)

// Resolution confidence thresholds for entity deduplication and conflict resolution.
// These thresholds determine when automatic resolution can be applied versus
// requiring manual review.
const (
	ResolutionConfidenceHigh   = 0.9 // High confidence, automatic resolution recommended
	ResolutionConfidenceMedium = 0.7 // Medium confidence, caution advised
	ResolutionConfidenceLow    = 0.5 // Low confidence, manual review recommended
)
