package cop

import "time"

func semLinkCompanionFleetFixture(now time.Time) CompanionFleet {
	generated := now.Add(-22 * time.Second).UTC()
	rawPolicy := "raw MAVLink frames stay on node-local telemetry ports; mesh summary index rejects raw packet payloads"
	return CompanionFleet{
		ID:                 "c360.edge.cop.semlink.fleet.simple-mesh-demo",
		Label:              "SemLink companion fleet",
		Source:             "semlink",
		Status:             "demo-ready",
		EvidenceKind:       "simple-mesh-companion-demo",
		VehicleProfile:     "ardurover-blueboat",
		NodeCount:          3,
		VehicleCount:       3,
		ExpectedSummaries:  3,
		AssertionState:     "passed",
		NoTransmitPosture:  "demo evidence only; no native transmit authority; no companion hardware transmit authority; raw MAVLink excluded from mesh summaries; SemOps exposes no mesh topology controls",
		RawMAVLinkExcluded: true,
		RawMAVLinkPolicy:   rawPolicy,
		DemoEvidenceLabel:  semLinkDemoEvidenceLabel,
		Confidence:         1,
		UpdatedAt:          generated,
		Nodes: []CompanionNode{
			{
				ID:                  "semlink-node-alpha",
				VehicleCount:        1,
				PeerCount:           2,
				InitialSummaryCount: 1,
				FinalSummaryCount:   3,
				WatermarkCount:      3,
				AppliedDiffCount:    2,
				DiffItemCount:       2,
				TTLMergePosture:     "bounded TTL summary merge; raw MAVLink frames excluded",
			},
			{
				ID:                  "semlink-node-bravo",
				VehicleCount:        1,
				PeerCount:           2,
				InitialSummaryCount: 1,
				FinalSummaryCount:   3,
				WatermarkCount:      3,
				AppliedDiffCount:    2,
				DiffItemCount:       2,
				TTLMergePosture:     "bounded TTL summary merge; raw MAVLink frames excluded",
			},
			{
				ID:                  "semlink-node-charlie",
				VehicleCount:        1,
				PeerCount:           2,
				InitialSummaryCount: 1,
				FinalSummaryCount:   3,
				WatermarkCount:      3,
				AppliedDiffCount:    2,
				DiffItemCount:       2,
				TTLMergePosture:     "bounded TTL summary merge; raw MAVLink frames excluded",
			},
		},
		Readback: CompanionReadback{
			AdapterStatus:            "unavailable",
			CommandACKStatus:         "unavailable",
			ResultStatus:             "unavailable",
			NativeExecutionAllowed:   false,
			CompanionTransmitAllowed: false,
		},
		CommandPosture: CompanionCommandPosture{
			Status: "unavailable",
		},
		RawMAVLinkExclusion: CompanionRawMAVLinkExclusion{
			RejectedBySummaryIndex: true,
			Policy:                 rawPolicy,
		},
		Assertions: []CompanionAssertion{
			{
				Name:   "node-count",
				Passed: true,
				Detail: "three companion nodes emitted summary evidence",
			},
			{
				Name:   "watermark-catch-up",
				Passed: true,
				Detail: "all nodes observed three watermarks after bounded diff catch-up",
			},
			{
				Name:   "raw-mavlink-exclusion",
				Passed: true,
				Detail: "summary replication excluded raw MAVLink payloads",
			},
		},
		Provenance: Provenance{
			Owner:     "semlink.e2e.demo",
			SourceRef: "semlink-demo://simple-mesh-companion-demo/fixture",
			Observed:  generated,
		},
	}
}
