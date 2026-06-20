package scenario

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	cotadapter "github.com/c360studio/semops/internal/adapters/cot"
	mavadapter "github.com/c360studio/semops/internal/adapters/mavlink"
	adsbprojector "github.com/c360studio/semops/internal/projectors/adsb"
	capprojector "github.com/c360studio/semops/internal/projectors/cap"
	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	capcodec "github.com/c360studio/semops/pkg/adapters/cap"
	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
)

const Phase1HADRScenarioID = "phase-1-hadr-flood-evacuation"

type State string

const (
	StateIdle      State = "idle"
	StateRunning   State = "running"
	StateSucceeded State = "succeeded"
	StateFailed    State = "failed"
)

type MAVLinkSink interface {
	IngestFrame(context.Context, []byte) (mavadapter.IngestResult, error)
	Health() mavadapter.Health
}

type CoTSink interface {
	IngestEvent(context.Context, []byte) (cotadapter.IngestResult, error)
	Health() cotadapter.Health
}

type CAPProjector interface {
	ProjectAlert(capcodec.Alert, string) (capprojector.Plan, error)
	MarkBornForPlan(capprojector.Plan) int
}

type CAPPlanWriter interface {
	Apply(context.Context, capprojector.Plan) error
}

type ADSBProjector interface {
	ProjectStates([]adsbprojector.SourceState) (adsbprojector.Plan, error)
	MarkBornForPlan(adsbprojector.Plan) int
}

type ADSBPlanWriter interface {
	Apply(context.Context, adsbprojector.Plan) error
}

type Config struct {
	Fixture       Fixture
	MAVLink       MAVLinkSink
	CoT           CoTSink
	CAPProjector  CAPProjector
	CAPWriter     CAPPlanWriter
	ADSBProjector ADSBProjector
	ADSBWriter    ADSBPlanWriter
	Clock         func() time.Time
}

type Runner struct {
	fixture       Fixture
	mavlink       MAVLinkSink
	cot           CoTSink
	capProjector  CAPProjector
	capWriter     CAPPlanWriter
	adsbProjector ADSBProjector
	adsbWriter    ADSBPlanWriter
	clock         func() time.Time

	mu     sync.RWMutex
	status Status
}

type Fixture struct {
	ID            string
	StartedAt     time.Time
	MAVLinkFrames []MAVLinkFrame
	CoTEvents     []CoTEvent
	CAPAlerts     []CAPAlert
	ADSBSnapshots []ADSBSnapshot
}

type MAVLinkFrame struct {
	Name   string
	Offset time.Duration
	Frame  []byte
}

type CoTEvent struct {
	Name   string
	Offset time.Duration
	RawXML []byte
}

type CAPAlert struct {
	Name   string
	Offset time.Duration
	Record capcodec.RawAlertRecord
}

type ADSBSnapshot struct {
	Name   string
	Offset time.Duration
	Record adsbcodec.RawSnapshotRecord
}

type Status struct {
	ScenarioID     string    `json:"scenario_id"`
	State          State     `json:"state"`
	CurrentStep    string    `json:"current_step,omitempty"`
	StartedAt      time.Time `json:"started_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
	FinishedAt     time.Time `json:"finished_at,omitempty"`
	CompletedSteps int       `json:"completed_steps"`
	FailedSteps    int       `json:"failed_steps"`
	LastError      string    `json:"last_error,omitempty"`
	Summary        Summary   `json:"summary"`
}

type Report struct {
	ScenarioID string       `json:"scenario_id"`
	State      State        `json:"state"`
	StartedAt  time.Time    `json:"started_at,omitempty"`
	FinishedAt time.Time    `json:"finished_at,omitempty"`
	Steps      []StepReport `json:"steps"`
	Summary    Summary      `json:"summary"`
	LastError  string       `json:"last_error,omitempty"`
}

type Summary struct {
	MAVLinkFrames int `json:"mavlink_frames"`
	CoTEvents     int `json:"cot_events"`
	CAPAlerts     int `json:"cap_alerts"`
	ADSBSnapshots int `json:"adsb_snapshots"`
	Mutations     int `json:"mutations"`
	Errors        int `json:"errors"`
}

type StepReport struct {
	Feed       string    `json:"feed"`
	Name       string    `json:"name"`
	RawRef     string    `json:"raw_ref,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
	Mutations  int       `json:"mutations"`
	Error      string    `json:"error,omitempty"`
}

func NewRunner(cfg Config) (*Runner, error) {
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	fixture := cfg.Fixture
	if fixture.ID == "" {
		generated, err := Phase1HADRFixture(cfg.Clock().UTC())
		if err != nil {
			return nil, err
		}
		fixture = generated
	}
	if err := fixture.Validate(); err != nil {
		return nil, err
	}
	runner := &Runner{
		fixture:       fixture,
		mavlink:       cfg.MAVLink,
		cot:           cfg.CoT,
		capProjector:  cfg.CAPProjector,
		capWriter:     cfg.CAPWriter,
		adsbProjector: cfg.ADSBProjector,
		adsbWriter:    cfg.ADSBWriter,
		clock:         cfg.Clock,
		status: Status{
			ScenarioID: fixture.ID,
			State:      StateIdle,
			UpdatedAt:  cfg.Clock().UTC(),
		},
	}
	return runner, nil
}

func Phase1HADRFixture(start time.Time) (Fixture, error) {
	if start.IsZero() {
		start = time.Now().UTC()
	}
	start = start.UTC()

	generator := mavcodec.NewGenerator(42, 7)
	heartbeat, err := generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{
		VehicleType:    mavcodec.TypeQuadrotor,
		Autopilot:      mavcodec.AutopilotPX4,
		BaseMode:       mavcodec.ModeFlagSafetyArmed,
		SystemStatus:   mavcodec.StateActive,
		MavlinkVersion: mavcodec.Version2,
	})
	if err != nil {
		return Fixture{}, fmt.Errorf("generate scenario MAVLink heartbeat: %w", err)
	}
	position, err := generator.GenerateGlobalPosition(mavcodec.PositionMessage{
		TimeBootMs:  1500,
		Lat:         389000001,
		Lon:         -770000002,
		Alt:         120000,
		RelativeAlt: 45000,
		Vx:          321,
		Vy:          -12,
		Vz:          7,
		Hdg:         27000,
	})
	if err != nil {
		return Fixture{}, fmt.Errorf("generate scenario MAVLink position: %w", err)
	}

	rawEvents, err := cotcodec.MarshalEvents(cotcodec.SeedEvents(start))
	if err != nil {
		return Fixture{}, fmt.Errorf("marshal scenario CoT seed events: %w", err)
	}
	cotEvents := make([]CoTEvent, 0, len(rawEvents))
	for i, raw := range rawEvents {
		cotEvents = append(cotEvents, CoTEvent{
			Name:   fmt.Sprintf("tak-cot-seed-%02d", i+1),
			Offset: (2 + time.Duration(i)) * time.Second,
			RawXML: append([]byte(nil), raw...),
		})
	}

	capRecords, err := capcodec.LifecycleFixtureRecords(start)
	if err != nil {
		return Fixture{}, fmt.Errorf("build scenario CAP lifecycle records: %w", err)
	}
	capAlerts := make([]CAPAlert, 0, len(capRecords))
	for i, record := range capRecords {
		capAlerts = append(capAlerts, CAPAlert{
			Name:   "cap-" + strings.TrimPrefix(record.Ref, "cap://fixture/hadr-flood/"),
			Offset: (10 + time.Duration(i)) * time.Second,
			Record: cloneCAPRecord(record),
		})
	}

	return Fixture{
		ID:        Phase1HADRScenarioID,
		StartedAt: start,
		MAVLinkFrames: []MAVLinkFrame{
			{Name: "mavlink-heartbeat", Frame: heartbeat},
			{Name: "mavlink-position", Offset: time.Second, Frame: position},
		},
		CoTEvents: cotEvents,
		CAPAlerts: capAlerts,
	}, nil
}

func (f Fixture) Validate() error {
	if f.ID == "" {
		return fmt.Errorf("scenario fixture requires an id")
	}
	for i, frame := range f.MAVLinkFrames {
		if frame.Name == "" {
			return fmt.Errorf("mavlink frame %d requires a name", i+1)
		}
		if len(frame.Frame) == 0 {
			return fmt.Errorf("mavlink frame %q is empty", frame.Name)
		}
	}
	for i, event := range f.CoTEvents {
		if event.Name == "" {
			return fmt.Errorf("cot event %d requires a name", i+1)
		}
		if len(event.RawXML) == 0 {
			return fmt.Errorf("cot event %q is empty", event.Name)
		}
	}
	for i, alert := range f.CAPAlerts {
		if alert.Name == "" {
			return fmt.Errorf("cap alert %d requires a name", i+1)
		}
		if alert.Record.Ref == "" {
			return fmt.Errorf("cap alert %q requires a source ref", alert.Name)
		}
		if len(alert.Record.RawXML) == 0 {
			return fmt.Errorf("cap alert %q is empty", alert.Name)
		}
	}
	for i, snapshot := range f.ADSBSnapshots {
		if snapshot.Name == "" {
			return fmt.Errorf("adsb snapshot %d requires a name", i+1)
		}
		if snapshot.Record.Ref == "" {
			return fmt.Errorf("adsb snapshot %q requires a source ref", snapshot.Name)
		}
		if len(snapshot.Record.RawJSON) == 0 {
			return fmt.Errorf("adsb snapshot %q is empty", snapshot.Name)
		}
	}
	return nil
}

func (r *Runner) Status() Status {
	if r == nil {
		return Status{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status
}

func (r *Runner) Run(ctx context.Context) (Report, error) {
	if r == nil {
		return Report{}, fmt.Errorf("scenario runner is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := r.requireSinks(); err != nil {
		return Report{}, err
	}

	started := r.now()
	report := Report{
		ScenarioID: r.fixture.ID,
		State:      StateRunning,
		StartedAt:  started,
	}
	r.setStatus(Status{
		ScenarioID: r.fixture.ID,
		State:      StateRunning,
		StartedAt:  started,
		UpdatedAt:  started,
	})

	for _, frame := range r.fixture.MAVLinkFrames {
		step, err := r.runMAVLinkStep(ctx, frame)
		report = appendStep(report, step)
		r.updateAfterStep(report, step, err)
		if err != nil {
			return r.failReport(report, err)
		}
	}
	for _, event := range r.fixture.CoTEvents {
		step, err := r.runCoTStep(ctx, event)
		report = appendStep(report, step)
		r.updateAfterStep(report, step, err)
		if err != nil {
			return r.failReport(report, err)
		}
	}
	for _, alert := range r.fixture.CAPAlerts {
		step, err := r.runCAPStep(ctx, alert)
		report = appendStep(report, step)
		r.updateAfterStep(report, step, err)
		if err != nil {
			return r.failReport(report, err)
		}
	}
	for _, snapshot := range r.fixture.ADSBSnapshots {
		step, err := r.runADSBStep(ctx, snapshot)
		report = appendStep(report, step)
		r.updateAfterStep(report, step, err)
		if err != nil {
			return r.failReport(report, err)
		}
	}

	finished := r.now()
	report.State = StateSucceeded
	report.FinishedAt = finished
	r.setStatus(Status{
		ScenarioID:     report.ScenarioID,
		State:          StateSucceeded,
		StartedAt:      report.StartedAt,
		UpdatedAt:      finished,
		FinishedAt:     finished,
		CompletedSteps: len(report.Steps),
		Summary:        report.Summary,
	})
	return report, nil
}

func (r *Runner) requireSinks() error {
	if len(r.fixture.MAVLinkFrames) > 0 && r.mavlink == nil {
		return fmt.Errorf("scenario %s includes MAVLink frames but no MAVLink sink", r.fixture.ID)
	}
	if len(r.fixture.CoTEvents) > 0 && r.cot == nil {
		return fmt.Errorf("scenario %s includes CoT events but no CoT sink", r.fixture.ID)
	}
	if len(r.fixture.CAPAlerts) > 0 {
		if r.capProjector == nil {
			return fmt.Errorf("scenario %s includes CAP alerts but no CAP projector", r.fixture.ID)
		}
		if r.capWriter == nil {
			return fmt.Errorf("scenario %s includes CAP alerts but no CAP graph writer", r.fixture.ID)
		}
	}
	if len(r.fixture.ADSBSnapshots) > 0 {
		if r.adsbProjector == nil {
			return fmt.Errorf("scenario %s includes ADS-B snapshots but no ADS-B projector", r.fixture.ID)
		}
		if r.adsbWriter == nil {
			return fmt.Errorf("scenario %s includes ADS-B snapshots but no ADS-B graph writer", r.fixture.ID)
		}
	}
	return nil
}

func (r *Runner) runMAVLinkStep(ctx context.Context, frame MAVLinkFrame) (StepReport, error) {
	step := r.startStep("mavlink", frame.Name)
	if err := ctx.Err(); err != nil {
		return r.finishStep(step, 0, "", err), err
	}
	result, err := r.mavlink.IngestFrame(ctx, frame.Frame)
	return r.finishStep(step, result.Mutations, result.RawRef, err), err
}

func (r *Runner) runCoTStep(ctx context.Context, event CoTEvent) (StepReport, error) {
	step := r.startStep("tak-cot", event.Name)
	if err := ctx.Err(); err != nil {
		return r.finishStep(step, 0, "", err), err
	}
	result, err := r.cot.IngestEvent(ctx, event.RawXML)
	return r.finishStep(step, result.Mutations, result.RawRef, err), err
}

func (r *Runner) runCAPStep(ctx context.Context, alert CAPAlert) (StepReport, error) {
	step := r.startStep("cap-edxl", alert.Name)
	if err := ctx.Err(); err != nil {
		return r.finishStep(step, 0, alert.Record.Ref, err), err
	}
	parsed, err := alert.Record.Alert()
	if err != nil {
		err = fmt.Errorf("parse CAP scenario alert %q: %w", alert.Name, err)
		return r.finishStep(step, 0, alert.Record.Ref, err), err
	}
	plan, err := r.capProjector.ProjectAlert(parsed, alert.Record.Ref)
	if err != nil {
		err = fmt.Errorf("project CAP scenario alert %q: %w", alert.Name, err)
		return r.finishStep(step, 0, alert.Record.Ref, err), err
	}
	if err := r.capWriter.Apply(ctx, plan); err != nil {
		err = fmt.Errorf("write CAP scenario alert %q: %w", alert.Name, err)
		return r.finishStep(step, len(plan.Mutations), alert.Record.Ref, err), err
	}
	r.capProjector.MarkBornForPlan(plan)
	return r.finishStep(step, len(plan.Mutations), alert.Record.Ref, nil), nil
}

func (r *Runner) runADSBStep(ctx context.Context, snapshot ADSBSnapshot) (StepReport, error) {
	step := r.startStep("adsb", snapshot.Name)
	if err := ctx.Err(); err != nil {
		return r.finishStep(step, 0, snapshot.Record.Ref, err), err
	}
	parsed, err := snapshot.Record.Snapshot()
	if err != nil {
		err = fmt.Errorf("parse ADS-B scenario snapshot %q: %w", snapshot.Name, err)
		return r.finishStep(step, 0, snapshot.Record.Ref, err), err
	}
	states := make([]adsbprojector.SourceState, 0, len(parsed.States))
	for _, state := range parsed.States {
		states = append(states, adsbprojector.SourceState{
			State:     state,
			SourceRef: snapshot.Record.Ref,
		})
	}
	plan, err := r.adsbProjector.ProjectStates(states)
	if err != nil {
		err = fmt.Errorf("project ADS-B scenario snapshot %q: %w", snapshot.Name, err)
		return r.finishStep(step, 0, snapshot.Record.Ref, err), err
	}
	if err := r.adsbWriter.Apply(ctx, plan); err != nil {
		err = fmt.Errorf("write ADS-B scenario snapshot %q: %w", snapshot.Name, err)
		return r.finishStep(step, len(plan.Mutations), snapshot.Record.Ref, err), err
	}
	r.adsbProjector.MarkBornForPlan(plan)
	return r.finishStep(step, len(plan.Mutations), snapshot.Record.Ref, nil), nil
}

func (r *Runner) startStep(feed, name string) StepReport {
	started := r.now()
	r.mu.Lock()
	r.status.State = StateRunning
	r.status.CurrentStep = feed + ":" + name
	r.status.UpdatedAt = started
	r.mu.Unlock()
	return StepReport{Feed: feed, Name: name, StartedAt: started}
}

func (r *Runner) finishStep(step StepReport, mutations int, rawRef string, err error) StepReport {
	step.FinishedAt = r.now()
	step.Mutations = mutations
	step.RawRef = rawRef
	if err != nil {
		step.Error = err.Error()
	}
	return step
}

func (r *Runner) failReport(report Report, err error) (Report, error) {
	finished := r.now()
	report.State = StateFailed
	report.FinishedAt = finished
	report.LastError = err.Error()
	r.setStatus(Status{
		ScenarioID:     report.ScenarioID,
		State:          StateFailed,
		CurrentStep:    "",
		StartedAt:      report.StartedAt,
		UpdatedAt:      finished,
		FinishedAt:     finished,
		CompletedSteps: successfulSteps(report.Steps),
		FailedSteps:    failedSteps(report.Steps),
		LastError:      report.LastError,
		Summary:        report.Summary,
	})
	return report, err
}

func (r *Runner) updateAfterStep(report Report, step StepReport, err error) {
	status := Status{
		ScenarioID:     report.ScenarioID,
		State:          StateRunning,
		CurrentStep:    step.Feed + ":" + step.Name,
		StartedAt:      report.StartedAt,
		UpdatedAt:      step.FinishedAt,
		CompletedSteps: successfulSteps(report.Steps),
		FailedSteps:    failedSteps(report.Steps),
		Summary:        report.Summary,
	}
	if err != nil {
		status.LastError = err.Error()
	}
	r.setStatus(status)
}

func (r *Runner) setStatus(status Status) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = status
}

func (r *Runner) now() time.Time {
	return r.clock().UTC()
}

func appendStep(report Report, step StepReport) Report {
	report.Steps = append(report.Steps, step)
	switch step.Feed {
	case "mavlink":
		report.Summary.MAVLinkFrames++
	case "tak-cot":
		report.Summary.CoTEvents++
	case "cap-edxl":
		report.Summary.CAPAlerts++
	case "adsb":
		report.Summary.ADSBSnapshots++
	}
	report.Summary.Mutations += step.Mutations
	if step.Error != "" {
		report.Summary.Errors++
	}
	return report
}

func successfulSteps(steps []StepReport) int {
	var total int
	for _, step := range steps {
		if step.Error == "" {
			total++
		}
	}
	return total
}

func failedSteps(steps []StepReport) int {
	var total int
	for _, step := range steps {
		if step.Error != "" {
			total++
		}
	}
	return total
}

func cloneCAPRecord(record capcodec.RawAlertRecord) capcodec.RawAlertRecord {
	record.RawXML = append([]byte(nil), record.RawXML...)
	return record
}
