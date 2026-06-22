package klv

import (
	"context"
	"testing"
	"time"

	klvprojector "github.com/c360studio/semops/internal/projectors/klv"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestKLVProjectorComponentWritesFrameGraphPlan(t *testing.T) {
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	writer := &recordingKLVPlanWriter{}
	projector := klvprojector.NewProjector(klvprojector.Config{
		OwnerTokens: map[string]ownership.OwnerToken{
			cop.OwnerKLV: ownership.ExpectedOwnerToken(cop.OwnerKLV, "component-test"),
		},
		TraceID: "component-klv-001",
	})
	component, err := NewProjectorComponent(ProjectorConfig{
		Registry:  registry,
		Projector: projector,
		Writer:    writer,
		Clock:     func() time.Time { return deterministicKLVFrameTime().Add(3 * time.Second) },
	})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}
	if err := component.Initialize(); err != nil {
		t.Fatalf("initialize projector: %v", err)
	}

	frame := deterministicKLVFramePayload()
	frameWire, err := marshalBaseMessage(MISB0601FrameType, frame, "semops-processor-klv-decode", frame.ReceivedAt)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}
	if err := component.HandleFrameMessage(context.Background(), frameWire); err != nil {
		t.Fatalf("handle frame message: %v", err)
	}
	if len(writer.plans) != 1 {
		t.Fatalf("plans = %d, want 1", len(writer.plans))
	}
	plan := writer.plans[0]
	if len(plan.Mutations) != 1 || plan.Mutations[0].Kind != klvprojector.MutationCreate {
		t.Fatalf("plan = %#v, want one create", plan)
	}
	create := plan.Mutations[0].Create
	if create.OwnerToken != "semops.feed.klv#component-test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.IndexingProfile != cop.KLVSensorFootprintContract().IndexingProfile {
		t.Fatalf("indexing profile = %q", create.IndexingProfile)
	}
	if create.TraceID != "component-klv-001" {
		t.Fatalf("trace id = %q", create.TraceID)
	}
	requireKLVTriple(t, create.Triples, cop.SensorFootprintSensorPosition, "POINT(-117.1234560 34.1234560)")
	requireKLVTriple(t, create.Triples, cop.SensorFootprintFrameCenter, "POINT(-117.1202220 34.1250010)")
	if got := component.DataFlow().MessagesPerSecond; got <= 0 {
		t.Fatalf("projector messages per second = %f, want > 0", got)
	}
}

type recordingKLVPlanWriter struct {
	plans []klvprojector.Plan
	err   error
}

func (w *recordingKLVPlanWriter) Apply(_ context.Context, plan klvprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return w.err
}

func deterministicKLVFramePayload() *MISB0601FramePayload {
	frame := NewMISB0601FramePayload(
		DefaultDecodeSource,
		"object://semops/klv/deterministic-001.ts",
		"klv://packet/deterministic/00000001",
		deterministicKLVFrameTime().Add(2*time.Second),
	)
	sensorLat, sensorLon := 34.123456, -117.123456
	sensorAlt := 1420.25
	azimuth, elevation := 87.5, -12.25
	centerLat, centerLon := 34.125001, -117.120222
	centerElevation := 905.5
	frame.FrameTime = deterministicKLVFrameTime()
	frame.PlatformDesignation = "SYNTHETIC-UAS-1"
	frame.SensorLatitude = &sensorLat
	frame.SensorLongitude = &sensorLon
	frame.SensorAltitudeMeters = &sensorAlt
	frame.SensorAzimuthDegrees = &azimuth
	frame.SensorElevationDegrees = &elevation
	frame.FrameCenterLatitude = &centerLat
	frame.FrameCenterLongitude = &centerLon
	frame.FrameCenterElevationMeters = &centerElevation
	frame.Fields = []string{
		"PrecisionTimeStamp",
		"PlatformDesignation",
		"SensorLatitude",
		"SensorLongitude",
		"SensorTrueAltitude",
		"SensorRelativeAzimuthAngle",
		"SensorRelativeElevationAngle",
		"FrameCenterLatitude",
		"FrameCenterLongitude",
		"FrameCenterElevation",
	}
	return frame
}

func deterministicKLVFrameTime() time.Time {
	return time.Date(2026, 6, 22, 17, 59, 58, 123456000, time.UTC)
}

func requireKLVTriple(t *testing.T, triples []message.Triple, predicate string, object any) {
	t.Helper()
	for _, triple := range triples {
		if triple.Predicate == predicate && triple.Object == object {
			return
		}
	}
	t.Fatalf("missing triple %s=%v in %#v", predicate, object, triples)
}
