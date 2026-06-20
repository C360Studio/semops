package scenario

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	adsbadapter "github.com/c360studio/semops/internal/adapters/adsb"
	cotadapter "github.com/c360studio/semops/internal/adapters/cot"
	adsbprojector "github.com/c360studio/semops/internal/projectors/adsb"
	capprojector "github.com/c360studio/semops/internal/projectors/cap"
	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	"github.com/c360studio/semops/internal/stack"
	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestRunnerReplaysPhase1HADRFixtureThroughAdapters(t *testing.T) {
	start := time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC)
	fixture, err := Phase1HADRFixture(start)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	adsbRecords, err := adsbcodec.OpenSkyFixtureRecords(start)
	if err != nil {
		t.Fatalf("adsb fixture records: %v", err)
	}
	for _, record := range adsbRecords {
		fixture.ADSBSnapshots = append(fixture.ADSBSnapshots, ADSBSnapshot{
			Name:   strings.TrimPrefix(record.Ref, "adsb://fixture/opensky-hadr/"),
			Offset: record.ReceivedAt.Sub(start),
			Record: cloneADSBRecord(record),
		})
	}

	tokens := testOwnerTokens("scenario")
	mavWriter := &recordingMAVLinkWriter{}
	mavAdapter, err := stack.NewMAVLinkAdapter(stack.MAVLinkAdapterConfig{
		Source:      "mavlink:scenario",
		Org:         "c360",
		Platform:    "scenario",
		OwnerTokens: tokens,
		TraceID:     "scenario-runner-test",
		Clock:       fixedClock(start),
	}, stack.MAVLinkAdapterDeps{Writer: mavWriter})
	if err != nil {
		t.Fatalf("mavlink adapter: %v", err)
	}

	cotWriter := &recordingCoTWriter{}
	cotAdapter, err := stack.NewCoTAdapter(stack.CoTAdapterConfig{
		Source:      "tak-cot:scenario",
		Org:         "c360",
		Platform:    "scenario",
		OwnerTokens: tokens,
		TraceID:     "scenario-runner-test",
		Clock:       fixedClock(start),
	}, stack.CoTAdapterDeps{Writer: cotWriter})
	if err != nil {
		t.Fatalf("cot adapter: %v", err)
	}

	capWriter := &recordingCAPWriter{}
	adsbWriter := &recordingADSBWriter{}
	adsbAdapter, err := stack.NewADSBAdapter(stack.ADSBAdapterConfig{
		Source:      "opensky-fixture",
		Org:         "c360",
		Platform:    "scenario",
		OwnerTokens: tokens,
		TraceID:     "scenario-runner-test",
		Clock:       fixedClock(start),
	}, stack.ADSBAdapterDeps{Writer: adsbWriter})
	if err != nil {
		t.Fatalf("adsb adapter: %v", err)
	}
	runner, err := NewRunner(Config{
		Fixture: fixture,
		MAVLink: mavAdapter,
		CoT:     cotAdapter,
		ADSB:    adsbAdapter,
		CAPProjector: capprojector.NewProjector(capprojector.Config{
			Org:         "c360",
			Platform:    "scenario",
			OwnerTokens: tokens,
			TraceID:     "scenario-runner-test",
		}),
		CAPWriter: capWriter,
		Clock:     fixedClock(start),
	})
	if err != nil {
		t.Fatalf("runner: %v", err)
	}

	report, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("run scenario: %v", err)
	}

	wantSteps := len(fixture.MAVLinkFrames) + len(fixture.CoTEvents) + len(fixture.CAPAlerts) + len(fixture.ADSBSnapshots)
	if report.State != StateSucceeded || len(report.Steps) != wantSteps {
		t.Fatalf("report state/steps = %s/%d, want %s/%d", report.State, len(report.Steps), StateSucceeded, wantSteps)
	}
	if report.Summary.MAVLinkFrames != 2 ||
		report.Summary.CoTEvents != 4 ||
		report.Summary.CAPAlerts != 4 ||
		report.Summary.ADSBSnapshots != 2 ||
		report.Summary.Errors != 0 {
		t.Fatalf("summary = %+v", report.Summary)
	}
	if report.Summary.Mutations == 0 {
		t.Fatal("expected scenario mutations")
	}
	if len(mavWriter.plans) != 2 {
		t.Fatalf("mavlink plans = %d, want heartbeat birth + position update", len(mavWriter.plans))
	}
	if len(cotWriter.plans) != 4 {
		t.Fatalf("cot plans = %d, want four seed event plans", len(cotWriter.plans))
	}
	if len(capWriter.plans) != 4 {
		t.Fatalf("cap plans = %d, want lifecycle create/update/update/create", len(capWriter.plans))
	}
	if len(adsbWriter.plans) != 2 {
		t.Fatalf("adsb plans = %d, want two OpenSky snapshot plans", len(adsbWriter.plans))
	}
	if report.Steps[len(report.Steps)-2].RawRef != "adsb://raw/opensky-fixture/00000001" ||
		report.Steps[len(report.Steps)-1].RawRef != "adsb://raw/opensky-fixture/00000002" {
		t.Fatalf("adsb raw refs = %q/%q, want captured raw lane refs",
			report.Steps[len(report.Steps)-2].RawRef,
			report.Steps[len(report.Steps)-1].RawRef)
	}
	if capWriter.plans[0].Mutations[0].Kind != capprojector.MutationCreate ||
		capWriter.plans[1].Mutations[0].Kind != capprojector.MutationUpdate ||
		capWriter.plans[2].Mutations[0].Kind != capprojector.MutationUpdate ||
		capWriter.plans[3].Mutations[0].Kind != capprojector.MutationCreate {
		t.Fatalf("cap mutation kinds = %s/%s/%s/%s",
			capWriter.plans[0].Mutations[0].Kind,
			capWriter.plans[1].Mutations[0].Kind,
			capWriter.plans[2].Mutations[0].Kind,
			capWriter.plans[3].Mutations[0].Kind)
	}
	if adsbWriter.plans[0].Mutations[0].Kind != adsbprojector.MutationCreate ||
		adsbWriter.plans[0].Mutations[1].Kind != adsbprojector.MutationCreate ||
		adsbWriter.plans[1].Mutations[0].Kind != adsbprojector.MutationUpdate ||
		adsbWriter.plans[1].Mutations[1].Kind != adsbprojector.MutationCreate {
		t.Fatalf("adsb mutation kinds = %s/%s/%s/%s",
			adsbWriter.plans[0].Mutations[0].Kind,
			adsbWriter.plans[0].Mutations[1].Kind,
			adsbWriter.plans[1].Mutations[0].Kind,
			adsbWriter.plans[1].Mutations[1].Kind)
	}

	status := runner.Status()
	if status.State != StateSucceeded ||
		status.CompletedSteps != wantSteps ||
		status.Summary.Mutations != report.Summary.Mutations {
		t.Fatalf("status = %+v; report = %+v", status, report.Summary)
	}
}

func TestRunnerFailsLoudlyWhenScenarioSinkIsMissing(t *testing.T) {
	fixture, err := Phase1HADRFixture(time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	runner, err := NewRunner(Config{Fixture: fixture, Clock: fixedClock(fixture.StartedAt)})
	if err != nil {
		t.Fatalf("runner: %v", err)
	}

	_, err = runner.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no MAVLink sink") {
		t.Fatalf("run error = %v, want missing MAVLink sink", err)
	}
	if status := runner.Status(); status.State != StateIdle {
		t.Fatalf("status = %+v, want idle after preflight failure", status)
	}
}

func TestRunnerStopsOnCAPWriteFailure(t *testing.T) {
	start := time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC)
	fullFixture, err := Phase1HADRFixture(start)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	fixture := Fixture{
		ID:        "cap-only-failure",
		StartedAt: start,
		CAPAlerts: fullFixture.CAPAlerts[:1],
	}
	writerErr := errors.New("graph unavailable")
	runner, err := NewRunner(Config{
		Fixture: fixture,
		CAPProjector: capprojector.NewProjector(capprojector.Config{
			Org:         "c360",
			Platform:    "scenario",
			OwnerTokens: testOwnerTokens("failure"),
		}),
		CAPWriter: &recordingCAPWriter{err: writerErr},
		Clock:     fixedClock(start),
	})
	if err != nil {
		t.Fatalf("runner: %v", err)
	}

	report, err := runner.Run(context.Background())
	if !errors.Is(err, writerErr) {
		t.Fatalf("run error = %v, want %v", err, writerErr)
	}
	if report.State != StateFailed ||
		report.Summary.CAPAlerts != 1 ||
		report.Summary.Errors != 1 ||
		report.Steps[0].Error == "" {
		t.Fatalf("failure report = %+v", report)
	}
	status := runner.Status()
	if status.State != StateFailed || status.FailedSteps != 1 || !strings.Contains(status.LastError, "cap-0001-alert") {
		t.Fatalf("status = %+v", status)
	}
}

type recordingMAVLinkWriter struct {
	plans []mavprojector.Plan
	err   error
}

func (w *recordingMAVLinkWriter) Apply(_ context.Context, plan mavprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return w.err
}

type recordingCoTWriter struct {
	plans []cotprojector.Plan
	err   error
}

func (w *recordingCoTWriter) Apply(_ context.Context, plan cotprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return w.err
}

type recordingCAPWriter struct {
	plans []capprojector.Plan
	err   error
}

func (w *recordingCAPWriter) Apply(_ context.Context, plan capprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return w.err
}

type recordingADSBWriter struct {
	plans []adsbprojector.Plan
	err   error
}

func (w *recordingADSBWriter) Apply(_ context.Context, plan adsbprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return w.err
}

func testOwnerTokens(suffix string) map[string]ownership.OwnerToken {
	return map[string]ownership.OwnerToken{
		cop.OwnerAsset:   ownership.ExpectedOwnerToken(cop.OwnerAsset, suffix),
		cop.OwnerMAVLink: ownership.ExpectedOwnerToken(cop.OwnerMAVLink, suffix),
		cop.OwnerTAK:     ownership.ExpectedOwnerToken(cop.OwnerTAK, suffix),
		cop.OwnerCAP:     ownership.ExpectedOwnerToken(cop.OwnerCAP, suffix),
		cop.OwnerADSB:    ownership.ExpectedOwnerToken(cop.OwnerADSB, suffix),
	}
}

func fixedClock(now time.Time) func() time.Time {
	return func() time.Time { return now }
}

func cloneADSBRecord(record adsbcodec.RawSnapshotRecord) adsbcodec.RawSnapshotRecord {
	record.RawJSON = append([]byte(nil), record.RawJSON...)
	return record
}

var _ cotadapter.PlanWriter = (*recordingCoTWriter)(nil)
var _ adsbadapter.PlanWriter = (*recordingADSBWriter)(nil)
