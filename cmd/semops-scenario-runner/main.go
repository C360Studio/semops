package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	semopsapp "github.com/c360studio/semops/internal/app"
	"github.com/c360studio/semops/internal/copownership"
	"github.com/c360studio/semops/internal/graphrequest"
	capprojector "github.com/c360studio/semops/internal/projectors/cap"
	"github.com/c360studio/semops/internal/scenario"
	"github.com/c360studio/semops/internal/stack"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

const (
	envScenarioAddr = "SEMOPS_SCENARIO_ADDR"
	defaultAddr     = ":8090"
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

	client, stopOwners, runner, err := composeRunner(ctx, cfg)
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

	done := make(chan error, 1)
	go func() {
		report, err := runner.Run(ctx)
		if err != nil {
			log.Printf("Scenario %s failed after %d steps: %v", report.ScenarioID, len(report.Steps), err)
			done <- err
			return
		}
		log.Printf(
			"Scenario %s succeeded: steps=%d mutations=%d mavlink=%d cot=%d cap=%d",
			report.ScenarioID,
			len(report.Steps),
			report.Summary.Mutations,
			report.Summary.MAVLinkFrames,
			report.Summary.CoTEvents,
			report.Summary.CAPAlerts,
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

	bindings, err := copownership.RegisterFirstPhase(ctx, registry, heartbeater)
	if err != nil {
		heartbeatCancel()
		_ = client.Close(context.Background())
		return nil, nil, nil, fmt.Errorf("register first-phase COP ownership: %w", err)
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

	runner, err := scenario.NewRunner(scenario.Config{
		MAVLink: mavlink,
		CoT:     cot,
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
	})
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
