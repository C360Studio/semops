package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	semopsapp "github.com/c360studio/semops/internal/app"
	"github.com/c360studio/semops/internal/copownership"
	"github.com/c360studio/semops/internal/graphrequest"
	capprojector "github.com/c360studio/semops/internal/projectors/cap"
	"github.com/c360studio/semops/internal/scenario"
	"github.com/c360studio/semops/internal/stack"
	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

const (
	envScenarioAddr           = "SEMOPS_SCENARIO_ADDR"
	envScenarioMode           = "SEMOPS_SCENARIO_MODE"
	envScenarioADSBFixture    = "SEMOPS_SCENARIO_ADSB_FIXTURE"
	envScenarioMAVLinkUDPAddr = "SEMOPS_SCENARIO_MAVLINK_UDP_ADDR"
	envScenarioCoTUDPAddr     = "SEMOPS_SCENARIO_COT_UDP_ADDR"
	envScenarioWriteTimeout   = "SEMOPS_SCENARIO_FEED_WRITE_TIMEOUT"
	envScenarioReadyURL       = "SEMOPS_SCENARIO_READY_URL"
	envScenarioReadyTimeout   = "SEMOPS_SCENARIO_READY_TIMEOUT"
	envScenarioCheckpoints    = "SEMOPS_SCENARIO_CHECKPOINT_MANIFEST"
	defaultAddr               = ":8090"
	defaultScenarioMode       = "product"
	scenarioModeProduct       = "product"
	scenarioModeContract      = "contract"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		log.Println("Received shutdown signal, terminating scenario runner...")
		cancel()
	}()

	log.Printf("SemOps scenario runner v%s (commit: %s, built: %s)", version, commit, buildDate)

	cfg, err := semopsapp.ConfigFromEnv(os.Getenv)
	if err != nil {
		log.Fatalf("Invalid SemOps scenario runner configuration: %v", err)
	}
	addr := scenarioAddr(os.Getenv)
	mode, err := scenarioModeFromEnv(os.Getenv)
	if err != nil {
		log.Fatalf("Invalid SemOps scenario runner configuration: %v", err)
	}
	includeADSB, err := scenarioADSBFixtureEnabled(os.Getenv)
	if err != nil {
		log.Fatalf("Invalid SemOps scenario runner configuration: %v", err)
	}
	boundary, err := scenarioFeedBoundaryFromEnv(os.Getenv)
	if err != nil {
		log.Fatalf("Invalid SemOps scenario runner configuration: %v", err)
	}
	readyURL := scenarioReadyURL(os.Getenv)
	readyTimeout, err := scenarioReadyTimeout(os.Getenv)
	if err != nil {
		log.Fatalf("Invalid SemOps scenario runner configuration: %v", err)
	}
	checkpoints, err := scenarioCheckpointManifestFromEnv(os.Getenv)
	if err != nil {
		log.Fatalf("Invalid SemOps scenario checkpoint configuration: %v", err)
	}

	client, stopOwners, runner, err := composeRunner(ctx, cfg, includeADSB, mode, boundary, checkpoints)
	if err != nil {
		log.Fatalf("Compose scenario runner: %v", err)
	}
	defer stopOwners()
	defer closeClient(client, cfg.ShutdownTimeout)

	server, err := startStatusServer(addr, runner)
	if err != nil {
		log.Fatalf("Start scenario status server: %v", err)
	}
	defer closeServer(server, cfg.ShutdownTimeout)

	if mode == scenarioModeProduct && readyURL != "" {
		log.Printf("Waiting for product feed target readiness: %s", readyURL)
		if err := waitHTTPReady(ctx, readyURL, readyTimeout); err != nil {
			log.Fatalf("Wait for product feed target readiness: %v", err)
		}
	}

	done := make(chan error, 1)
	go func() {
		report, err := runner.Run(ctx)
		if err != nil {
			log.Printf("Scenario %s failed after %d steps: %v", report.ScenarioID, len(report.Steps), err)
			done <- err
			return
		}
		log.Printf(
			"Scenario %s succeeded: ingress=%s steps=%d feed_boundary_deliveries=%d mutations=%d mavlink=%d cot=%d cap=%d adsb=%d",
			report.ScenarioID,
			report.IngressMode,
			len(report.Steps),
			report.Summary.FeedBoundaryDeliveries,
			report.Summary.Mutations,
			report.Summary.MAVLinkFrames,
			report.Summary.CoTEvents,
			report.Summary.CAPAlerts,
			report.Summary.ADSBSnapshots,
		)
		done <- nil
	}()

	select {
	case <-ctx.Done():
	case <-done:
		<-ctx.Done()
	}
}

func composeRunner(
	ctx context.Context,
	cfg semopsapp.Config,
	includeADSB bool,
	mode string,
	boundary scenarioFeedBoundary,
	checkpoints scenario.CheckpointManifest,
) (*natsclient.Client, func(), *scenario.Runner, error) {
	switch mode {
	case scenarioModeProduct:
		return composeProductRunner(cfg, includeADSB, boundary, checkpoints)
	case scenarioModeContract:
		return composeContractRunner(ctx, cfg, includeADSB, checkpoints)
	default:
		return nil, nil, nil, fmt.Errorf("unsupported scenario mode %q", mode)
	}
}

func composeProductRunner(
	cfg semopsapp.Config,
	includeADSB bool,
	boundary scenarioFeedBoundary,
	checkpoints scenario.CheckpointManifest,
) (*natsclient.Client, func(), *scenario.Runner, error) {
	if includeADSB {
		return nil, nil, nil, fmt.Errorf("%s=true is not supported in product mode; use hosted ADS-B components", envScenarioADSBFixture)
	}
	fixture, err := scenarioProductFixture(time.Now().UTC())
	if err != nil {
		return nil, nil, nil, err
	}
	mavlinkSink, err := scenario.NewMAVLinkUDPSink(scenario.UDPFeedSinkConfig{
		Addr:         boundary.MAVLinkUDPAddr,
		Source:       cfg.MAVLink.Source,
		WriteTimeout: boundary.WriteTimeout,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("compose MAVLink feed-boundary sink: %w", err)
	}
	cotSink, err := scenario.NewCoTUDPSink(scenario.UDPFeedSinkConfig{
		Addr:         boundary.CoTUDPAddr,
		Source:       cfg.CoT.Source,
		WriteTimeout: boundary.WriteTimeout,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("compose CoT feed-boundary sink: %w", err)
	}
	runner, err := scenario.NewRunner(scenario.Config{
		Fixture:     fixture,
		MAVLink:     mavlinkSink,
		CoT:         cotSink,
		IngressMode: scenario.IngressModeFeedBoundary,
		Checkpoints: checkpoints,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create product scenario runner: %w", err)
	}
	return nil, func() {}, runner, nil
}

func composeContractRunner(
	ctx context.Context,
	cfg semopsapp.Config,
	includeADSB bool,
	checkpoints scenario.CheckpointManifest,
) (*natsclient.Client, func(), *scenario.Runner, error) {
	client, err := natsclient.NewClient(
		cfg.NATSURL,
		natsclient.WithName(cfg.NATSName),
		natsclient.WithTimeout(cfg.NATSConnectTimeout),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create SemStreams NATS client: %w", err)
	}
	if err := client.Connect(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("connect SemStreams NATS: %w", err)
	}

	registry, err := ownership.EnsureBuckets(ctx, client, nil, nil)
	if err != nil {
		_ = client.Close(context.Background())
		return nil, nil, nil, fmt.Errorf("ensure ownership buckets: %w", err)
	}
	heartbeater := registry.NewHeartbeater(cfg.OwnershipHeartbeatInterval)
	heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())
	go heartbeater.Run(heartbeatCtx)

	bindings, err := copownership.RegisterOwnedContracts(
		ctx,
		registry,
		heartbeater,
		scenarioOwnedContracts(includeADSB),
	)
	if err != nil {
		heartbeatCancel()
		_ = client.Close(context.Background())
		return nil, nil, nil, fmt.Errorf("register scenario COP ownership: %w", err)
	}

	requester := graphrequest.NewNATSRequester(client, graphrequest.WithRetryConfig(cfg.MAVLink.Retry))
	mavlink, err := stack.NewMAVLinkAdapter(stack.MAVLinkAdapterConfig{
		Source:        cfg.MAVLink.Source,
		Org:           cfg.MAVLink.Org,
		Platform:      cfg.MAVLink.Platform,
		OwnerTokens:   bindings.OwnerTokenMap(),
		TraceID:       cfg.MAVLink.TraceID,
		RawMaxRecords: cfg.MAVLink.RawMaxRecords,
		RawMaxBytes:   cfg.MAVLink.RawMaxBytes,
		WriteTimeout:  cfg.MAVLink.WriteTimeout,
		Retry:         cfg.MAVLink.Retry,
	}, stack.MAVLinkAdapterDeps{NATS: client})
	if err != nil {
		heartbeatCancel()
		_ = client.Close(context.Background())
		return nil, nil, nil, fmt.Errorf("compose MAVLink adapter: %w", err)
	}
	cot, err := stack.NewCoTAdapter(stack.CoTAdapterConfig{
		Source:        cfg.CoT.Source,
		Org:           cfg.CoT.Org,
		Platform:      cfg.CoT.Platform,
		OwnerTokens:   bindings.OwnerTokenMap(),
		TraceID:       cfg.CoT.TraceID,
		RawMaxRecords: cfg.CoT.RawMaxRecords,
		RawMaxBytes:   cfg.CoT.RawMaxBytes,
		WriteTimeout:  cfg.CoT.WriteTimeout,
		Retry:         cfg.CoT.Retry,
	}, stack.CoTAdapterDeps{NATS: client})
	if err != nil {
		heartbeatCancel()
		_ = client.Close(context.Background())
		return nil, nil, nil, fmt.Errorf("compose CoT adapter: %w", err)
	}

	fixture, err := scenarioFixture(time.Now().UTC(), includeADSB)
	if err != nil {
		heartbeatCancel()
		_ = client.Close(context.Background())
		return nil, nil, nil, err
	}

	runnerConfig := scenario.Config{
		Fixture:     fixture,
		MAVLink:     mavlink,
		CoT:         cot,
		IngressMode: scenario.IngressModeDirectGraphContract,
		Checkpoints: checkpoints,
		CAPProjector: capprojector.NewProjector(capprojector.Config{
			Org:         cfg.MAVLink.Org,
			Platform:    cfg.MAVLink.Platform,
			OwnerTokens: bindings.OwnerTokenMap(),
			TraceID:     cfg.MAVLink.TraceID,
		}),
		CAPWriter: capprojector.NewGraphWriter(
			requester,
			capprojector.WithWriteTimeout(cfg.MAVLink.WriteTimeout),
		),
	}
	if includeADSB {
		adsb, err := stack.NewADSBAdapter(stack.ADSBAdapterConfig{
			Source:        "opensky-fixture",
			Org:           cfg.MAVLink.Org,
			Platform:      cfg.MAVLink.Platform,
			OwnerTokens:   bindings.OwnerTokenMap(),
			TraceID:       "semops-adsb-scenario-runner",
			RawMaxRecords: cfg.MAVLink.RawMaxRecords,
			RawMaxBytes:   cfg.MAVLink.RawMaxBytes,
			WriteTimeout:  cfg.MAVLink.WriteTimeout,
			Retry:         cfg.MAVLink.Retry,
		}, stack.ADSBAdapterDeps{NATS: client})
		if err != nil {
			heartbeatCancel()
			_ = client.Close(context.Background())
			return nil, nil, nil, fmt.Errorf("compose ADS-B adapter: %w", err)
		}
		runnerConfig.ADSB = adsb
	}

	runner, err := scenario.NewRunner(runnerConfig)
	if err != nil {
		heartbeatCancel()
		_ = client.Close(context.Background())
		return nil, nil, nil, fmt.Errorf("create scenario runner: %w", err)
	}

	return client, heartbeatCancel, runner, nil
}

func startStatusServer(addr string, runner *scenario.Runner) (*http.Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           scenario.NewStatusHandler(runner.Status),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("SemOps scenario status listening on %s", listener.Addr())
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("SemOps scenario status server exited: %v", err)
		}
	}()
	return server, nil
}

func scenarioFixture(start time.Time, includeADSB bool) (scenario.Fixture, error) {
	fixture, err := scenario.Phase1HADRFixture(start)
	if err != nil {
		return scenario.Fixture{}, fmt.Errorf("build scenario fixture: %w", err)
	}
	if !includeADSB {
		return fixture, nil
	}
	records, err := adsbcodec.OpenSkyFixtureRecords(fixture.StartedAt)
	if err != nil {
		return scenario.Fixture{}, fmt.Errorf("build ADS-B scenario fixture records: %w", err)
	}
	for _, record := range records {
		fixture.ADSBSnapshots = append(fixture.ADSBSnapshots, scenario.ADSBSnapshot{
			Name:   strings.TrimPrefix(record.Ref, "adsb://fixture/opensky-hadr/"),
			Offset: record.ReceivedAt.Sub(fixture.StartedAt),
			Record: cloneADSBRecord(record),
		})
	}
	return fixture, nil
}

func scenarioProductFixture(start time.Time) (scenario.Fixture, error) {
	fixture, err := scenarioFixture(start, false)
	if err != nil {
		return scenario.Fixture{}, err
	}
	fixture.CAPAlerts = nil
	fixture.ADSBSnapshots = nil
	return fixture, nil
}

func scenarioOwnedContracts(includeADSB bool) []cop.OwnedContract {
	owned := cop.FirstPhaseOwnedContracts()
	if includeADSB {
		owned = append(owned, cop.OwnedContract{
			Owner:    cop.OwnerADSB,
			Contract: cop.ADSBTrackContract(),
		})
	}
	return owned
}

func closeServer(server *http.Server, timeout time.Duration) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Close scenario status server: %v", err)
	}
}

func closeClient(client *natsclient.Client, timeout time.Duration) {
	if client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := client.Close(ctx); err != nil {
		log.Printf("Close SemStreams NATS client: %v", err)
	}
}

func scenarioAddr(getenv func(string) string) string {
	value := strings.TrimSpace(getenv(envScenarioAddr))
	if value == "" {
		return defaultAddr
	}
	return value
}

type scenarioFeedBoundary struct {
	MAVLinkUDPAddr string
	CoTUDPAddr     string
	WriteTimeout   time.Duration
}

func scenarioModeFromEnv(getenv func(string) string) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	value := strings.TrimSpace(getenv(envScenarioMode))
	if value == "" {
		return defaultScenarioMode, nil
	}
	switch strings.ToLower(value) {
	case scenarioModeProduct:
		return scenarioModeProduct, nil
	case scenarioModeContract:
		return scenarioModeContract, nil
	default:
		return "", fmt.Errorf("%s must be %q or %q", envScenarioMode, scenarioModeProduct, scenarioModeContract)
	}
}

func scenarioFeedBoundaryFromEnv(getenv func(string) string) (scenarioFeedBoundary, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	cfg := scenarioFeedBoundary{
		MAVLinkUDPAddr: firstNonEmpty(strings.TrimSpace(getenv(envScenarioMAVLinkUDPAddr)), "127.0.0.1:14550"),
		CoTUDPAddr:     firstNonEmpty(strings.TrimSpace(getenv(envScenarioCoTUDPAddr)), "127.0.0.1:18090"),
		WriteTimeout:   time.Second,
	}
	if value := strings.TrimSpace(getenv(envScenarioWriteTimeout)); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return scenarioFeedBoundary{}, fmt.Errorf("parse %s: %w", envScenarioWriteTimeout, err)
		}
		if duration <= 0 {
			return scenarioFeedBoundary{}, fmt.Errorf("%s must be greater than zero", envScenarioWriteTimeout)
		}
		cfg.WriteTimeout = duration
	}
	if cfg.MAVLinkUDPAddr == "" {
		return scenarioFeedBoundary{}, fmt.Errorf("%s is required in product mode", envScenarioMAVLinkUDPAddr)
	}
	if cfg.CoTUDPAddr == "" {
		return scenarioFeedBoundary{}, fmt.Errorf("%s is required in product mode", envScenarioCoTUDPAddr)
	}
	return cfg, nil
}

func scenarioReadyURL(getenv func(string) string) string {
	if getenv == nil {
		getenv = os.Getenv
	}
	return strings.TrimSpace(getenv(envScenarioReadyURL))
}

func scenarioReadyTimeout(getenv func(string) string) (time.Duration, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	value := strings.TrimSpace(getenv(envScenarioReadyTimeout))
	if value == "" {
		return 60 * time.Second, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", envScenarioReadyTimeout, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", envScenarioReadyTimeout)
	}
	return duration, nil
}

func scenarioCheckpointManifestFromEnv(getenv func(string) string) (scenario.CheckpointManifest, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	path := strings.TrimSpace(getenv(envScenarioCheckpoints))
	if path == "" {
		return scenario.CheckpointManifest{}, nil
	}
	manifest, err := scenario.LoadCheckpointManifest(path)
	if err != nil {
		return scenario.CheckpointManifest{}, fmt.Errorf("%s: %w", envScenarioCheckpoints, err)
	}
	return manifest, nil
}

func scenarioADSBFixtureEnabled(getenv func(string) string) (bool, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	value := strings.TrimSpace(getenv(envScenarioADSBFixture))
	if value == "" {
		return false, nil
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", envScenarioADSBFixture, err)
	}
	return enabled, nil
}

func waitHTTPReady(ctx context.Context, readyURL string, timeout time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if readyURL == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		req, err := http.NewRequestWithContext(waitCtx, http.MethodGet, readyURL, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		}
		select {
		case <-waitCtx.Done():
			if lastErr != nil {
				return fmt.Errorf("timed out waiting for %s: %w", readyURL, lastErr)
			}
			return fmt.Errorf("timed out waiting for %s: %w", readyURL, waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cloneADSBRecord(record adsbcodec.RawSnapshotRecord) adsbcodec.RawSnapshotRecord {
	record.RawJSON = append([]byte(nil), record.RawJSON...)
	return record
}
