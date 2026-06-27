package app

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/c360studio/semops/internal/componentmetrics"
	adsbcomponent "github.com/c360studio/semops/internal/components/adsb"
	capcomponent "github.com/c360studio/semops/internal/components/cap"
	cotcomponent "github.com/c360studio/semops/internal/components/cot"
	fusioncomponent "github.com/c360studio/semops/internal/components/fusion"
	klvcomponent "github.com/c360studio/semops/internal/components/klv"
	mavcomponent "github.com/c360studio/semops/internal/components/mavlink"
	sapientcomponent "github.com/c360studio/semops/internal/components/sapient"
	weathercomponent "github.com/c360studio/semops/internal/components/weather"
	"github.com/c360studio/semops/internal/copownership"
	fusionassociation "github.com/c360studio/semops/internal/fusion/association"
	"github.com/c360studio/semops/internal/graphrequest"
	adsbprojector "github.com/c360studio/semops/internal/projectors/adsb"
	capprojector "github.com/c360studio/semops/internal/projectors/cap"
	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	fusionprojector "github.com/c360studio/semops/internal/projectors/fusion"
	klvprojector "github.com/c360studio/semops/internal/projectors/klv"
	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	sapientprojector "github.com/c360studio/semops/internal/projectors/sapient"
	weatherprojector "github.com/c360studio/semops/internal/projectors/weather"
	"github.com/c360studio/semops/internal/stack"
	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	capcodec "github.com/c360studio/semops/pkg/adapters/cap"
	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	sapientcodec "github.com/c360studio/semops/pkg/adapters/sapient"
	weathercodec "github.com/c360studio/semops/pkg/adapters/weather"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/nats-io/nats.go"
)

func TestStartRegistersOwnershipBeforeComposingMAVLinkFlow(t *testing.T) {
	ctx := context.Background()
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.NATSURL = "nats://semstreams:4222"
	cfg.MAVLink.Platform = "edge-alpha"
	cfg.MAVLink.Source = "udp:14550"
	cfg.MAVLink.WriteTimeout = 750 * time.Millisecond

	var stoppedOwners bool
	var gotWriterCfg stack.MAVLinkAdapterConfig
	var gotWriterDeps stack.MAVLinkAdapterDeps

	app, err := start(ctx, cfg, dependencies{
		newNATSClient: func(got Config) (semstreamsClient, error) {
			if got.NATSURL != cfg.NATSURL {
				t.Fatalf("NATS URL = %q, want %q", got.NATSURL, cfg.NATSURL)
			}
			return client, nil
		},
		registerOwners: func(
			_ context.Context,
			gotClient semstreamsClient,
			heartbeat time.Duration,
			_ []cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			if gotClient != client {
				t.Fatal("ownership registration got unexpected client")
			}
			if !client.connected {
				t.Fatal("ownership registration must run after NATS connect")
			}
			if heartbeat != cfg.OwnershipHeartbeatInterval {
				t.Fatalf("heartbeat interval = %s, want %s", heartbeat, cfg.OwnershipHeartbeatInterval)
			}
			return copownership.BindingResult{
					Incarnation: "lease-123",
					Owners:      []string{cop.OwnerMAVLink},
					Tokens: map[string]ownership.OwnerToken{
						cop.OwnerAsset:   ownership.ExpectedOwnerToken(cop.OwnerAsset, "lease-123"),
						cop.OwnerMAVLink: ownership.ExpectedOwnerToken(cop.OwnerMAVLink, "lease-123"),
					},
				}, func() {
					stoppedOwners = true
				}, nil
		},
		newMAVLinkPlanWriter: func(
			writerCfg stack.MAVLinkAdapterConfig,
			writerDeps stack.MAVLinkAdapterDeps,
		) (mavcomponent.PlanWriter, error) {
			gotWriterCfg = writerCfg
			gotWriterDeps = writerDeps
			return &recordingAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.OwnershipBinding().Incarnation != "lease-123" {
		t.Fatalf("ownership incarnation = %q", app.OwnershipBinding().Incarnation)
	}
	if app.MAVLinkDecoder() == nil || app.MAVLinkProjector() == nil {
		t.Fatal("expected hosted MAVLink decoder and projector components")
	}
	if got, want := gotWriterCfg.OwnerTokens[cop.OwnerMAVLink].Wire(), "semops.feed.mavlink#lease-123"; got != want {
		t.Fatalf("writer MAVLink owner token = %q, want %q", got, want)
	}
	if gotWriterCfg.Platform != "edge-alpha" || gotWriterCfg.Source != "udp:14550" {
		t.Fatalf("writer config = %+v", gotWriterCfg)
	}
	if gotWriterCfg.WriteTimeout != 750*time.Millisecond {
		t.Fatalf("writer timeout = %s", gotWriterCfg.WriteTimeout)
	}
	if gotWriterDeps.NATS != client {
		t.Fatal("MAVLink writer must reuse the connected SemStreams client")
	}
	if !client.hasSubscription(mavcomponent.DefaultDecodedSubject) ||
		!client.hasSubscription(mavcomponent.DefaultRawSubject) {
		t.Fatalf("MAVLink flow subscriptions = %+v", client.subscriptionSubjects())
	}

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
	if !stoppedOwners {
		t.Fatal("close must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("close must close SemStreams client")
	}
}

func TestStartCleansUpWhenMAVLinkCompositionFails(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	adapterErr := errors.New("bad adapter")
	var stoppedOwners bool

	_, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {
				stoppedOwners = true
			}, nil
		},
		newMAVLinkPlanWriter: func(
			stack.MAVLinkAdapterConfig,
			stack.MAVLinkAdapterDeps,
		) (mavcomponent.PlanWriter, error) {
			return nil, adapterErr
		},
	})
	if !errors.Is(err, adapterErr) {
		t.Fatalf("error = %v, want adapter error", err)
	}
	if !stoppedOwners {
		t.Fatal("failed startup must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("failed startup must close SemStreams client")
	}
}

func TestStartRegistersOwnershipBeforeComposingCoTFlow(t *testing.T) {
	ctx := context.Background()
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.CoT.Enabled = true
	cfg.CoT.Platform = "edge-alpha"
	cfg.CoT.Source = "udp:cot"
	cfg.CoT.WriteTimeout = 750 * time.Millisecond

	var stoppedOwners bool
	var gotWriterCfg stack.CoTAdapterConfig
	var gotWriterDeps stack.CoTAdapterDeps

	app, err := start(ctx, cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			_ context.Context,
			gotClient semstreamsClient,
			heartbeat time.Duration,
			_ []cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			if gotClient != client {
				t.Fatal("ownership registration got unexpected client")
			}
			if !client.connected {
				t.Fatal("ownership registration must run after NATS connect")
			}
			if heartbeat != cfg.OwnershipHeartbeatInterval {
				t.Fatalf("heartbeat interval = %s, want %s", heartbeat, cfg.OwnershipHeartbeatInterval)
			}
			return copownership.BindingResult{
					Incarnation: "lease-123",
					Owners:      []string{cop.OwnerTAK},
					Tokens: map[string]ownership.OwnerToken{
						cop.OwnerAsset: ownership.ExpectedOwnerToken(cop.OwnerAsset, "lease-123"),
						cop.OwnerTAK:   ownership.ExpectedOwnerToken(cop.OwnerTAK, "lease-123"),
					},
				}, func() {
					stoppedOwners = true
				}, nil
		},
		newCoTPlanWriter: func(
			writerCfg stack.CoTAdapterConfig,
			writerDeps stack.CoTAdapterDeps,
		) (cotcomponent.PlanWriter, error) {
			gotWriterCfg = writerCfg
			gotWriterDeps = writerDeps
			return &recordingCoTAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.CoTDecoder() == nil || app.CoTProjector() == nil {
		t.Fatal("expected hosted CoT decoder and projector components")
	}
	if got, want := gotWriterCfg.OwnerTokens[cop.OwnerTAK].Wire(), "semops.feed.tak#lease-123"; got != want {
		t.Fatalf("writer TAK owner token = %q, want %q", got, want)
	}
	if gotWriterCfg.Platform != "edge-alpha" || gotWriterCfg.Source != "udp:cot" {
		t.Fatalf("writer config = %+v", gotWriterCfg)
	}
	if gotWriterCfg.WriteTimeout != 750*time.Millisecond {
		t.Fatalf("writer timeout = %s", gotWriterCfg.WriteTimeout)
	}
	if gotWriterDeps.NATS != client {
		t.Fatal("CoT writer must reuse the connected SemStreams client")
	}
	if !client.hasSubscription(cotcomponent.DefaultDecodedSubject) ||
		!client.hasSubscription(cotcomponent.DefaultRawSubject) {
		t.Fatalf("CoT flow subscriptions = %+v", client.subscriptionSubjects())
	}

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
	if !stoppedOwners {
		t.Fatal("close must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("close must close SemStreams client")
	}
}

func TestStartCleansUpWhenCoTCompositionFails(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.CoT.Enabled = true
	adapterErr := errors.New("bad cot writer")
	var stoppedOwners bool

	_, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {
				stoppedOwners = true
			}, nil
		},
		newCoTPlanWriter: func(
			stack.CoTAdapterConfig,
			stack.CoTAdapterDeps,
		) (cotcomponent.PlanWriter, error) {
			return nil, adapterErr
		},
	})
	if !errors.Is(err, adapterErr) {
		t.Fatalf("error = %v, want adapter error", err)
	}
	if !stoppedOwners {
		t.Fatal("failed startup must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("failed startup must close SemStreams client")
	}
}

func TestStartRegistersOwnershipBeforeComposingCAPFlow(t *testing.T) {
	ctx := context.Background()
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.CAP.Enabled = true
	cfg.CAP.Platform = "edge-alpha"
	cfg.CAP.Source = "cap:http:test"
	cfg.CAP.WriteTimeout = 750 * time.Millisecond
	cfg.CAP.HTTP.URL = "https://example.test/cap"
	cfg.CAP.HTTP.PollInterval = time.Hour
	cfg.CAP.HTTP.StaleAfter = 3 * time.Hour
	cfg.CAP.HTTP.ContactPolicy = "semops-test@example.invalid"

	var stoppedOwners bool
	var gotWriterCfg stack.CAPAdapterConfig
	var gotWriterDeps stack.CAPAdapterDeps

	app, err := start(ctx, cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			_ context.Context,
			gotClient semstreamsClient,
			heartbeat time.Duration,
			_ []cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			if gotClient != client {
				t.Fatal("ownership registration got unexpected client")
			}
			if !client.connected {
				t.Fatal("ownership registration must run after NATS connect")
			}
			if heartbeat != cfg.OwnershipHeartbeatInterval {
				t.Fatalf("heartbeat interval = %s, want %s", heartbeat, cfg.OwnershipHeartbeatInterval)
			}
			return copownership.BindingResult{
					Incarnation: "lease-123",
					Owners:      []string{cop.OwnerCAP},
					Tokens: map[string]ownership.OwnerToken{
						cop.OwnerCAP: ownership.ExpectedOwnerToken(cop.OwnerCAP, "lease-123"),
					},
				}, func() {
					stoppedOwners = true
				}, nil
		},
		newCAPPlanWriter: func(
			writerCfg stack.CAPAdapterConfig,
			writerDeps stack.CAPAdapterDeps,
		) (capcomponent.PlanWriter, error) {
			gotWriterCfg = writerCfg
			gotWriterDeps = writerDeps
			return &recordingCAPAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.CAPHTTPPoller() == nil || app.CAPDecoder() == nil || app.CAPProjector() == nil {
		t.Fatal("expected hosted CAP poller, decoder, and projector components")
	}
	if got, want := gotWriterCfg.OwnerTokens[cop.OwnerCAP].Wire(), "semops.feed.cap#lease-123"; got != want {
		t.Fatalf("writer CAP owner token = %q, want %q", got, want)
	}
	if gotWriterCfg.Platform != "edge-alpha" || gotWriterCfg.Source != "cap:http:test" {
		t.Fatalf("writer config = %+v", gotWriterCfg)
	}
	if gotWriterCfg.WriteTimeout != 750*time.Millisecond {
		t.Fatalf("writer timeout = %s", gotWriterCfg.WriteTimeout)
	}
	if gotWriterDeps.NATS != client {
		t.Fatal("CAP writer must reuse the connected SemStreams client")
	}
	if got := app.CAPHTTPPoller().DebugStatus().(capcomponent.HTTPPollerDebugStatus).StaleAfter; got != 3*time.Hour {
		t.Fatalf("CAP poller stale_after = %s, want 3h", got)
	}
	if !client.hasSubscription(capcomponent.DefaultDecodedSubject) ||
		!client.hasSubscription(capcomponent.DefaultRawSubject) {
		t.Fatalf("CAP flow subscriptions = %+v", client.subscriptionSubjects())
	}

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
	if !stoppedOwners {
		t.Fatal("close must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("close must close SemStreams client")
	}
}

func TestStartCAPFlowCapturesProviderReplayWhenConfigured(t *testing.T) {
	now := time.Date(2026, 6, 20, 18, 0, 0, 0, time.UTC)
	records, err := capcodec.LifecycleFixtureRecords(now)
	if err != nil {
		t.Fatalf("cap fixtures: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/cap+xml")
		_, _ = w.Write(records[0].RawXML)
	}))
	defer server.Close()

	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.CAP.Enabled = true
	cfg.CAP.Source = "cap:http:runtime-provider-fixture"
	cfg.CAP.ReplayPath = filepath.Join(t.TempDir(), "cap-provider.jsonl")
	cfg.CAP.HTTP.URL = server.URL
	cfg.CAP.HTTP.PollInterval = time.Hour
	writer := &recordingCAPAppPlanWriter{}

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{
				Incarnation: "lease-123",
				Owners:      []string{cop.OwnerCAP},
				Tokens: map[string]ownership.OwnerToken{
					cop.OwnerCAP: ownership.ExpectedOwnerToken(cop.OwnerCAP, "lease-123"),
				},
			}, func() {}, nil
		},
		newCAPPlanWriter: func(
			stack.CAPAdapterConfig,
			stack.CAPAdapterDeps,
		) (capcomponent.PlanWriter, error) {
			return writer, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.Close(context.Background()); err != nil {
			t.Fatalf("close app: %v", err)
		}
	})

	if err := app.CAPHTTPPoller().PollOnce(context.Background()); err != nil {
		t.Fatalf("poll CAP provider fixture: %v", err)
	}
	loaded, err := capcodec.LoadReplay(cfg.CAP.ReplayPath)
	if err != nil {
		t.Fatalf("load runtime CAP replay: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("runtime CAP replay records = %d, want 1", len(loaded))
	}
	if loaded[0].Source != "cap:http:runtime-provider-fixture" ||
		loaded[0].Identifier != "nws-demo-flood-warning" {
		t.Fatalf("runtime CAP replay record = %+v", loaded[0])
	}
	if len(writer.plans) != 1 {
		t.Fatalf("CAP graph plans = %d, want hazard projection", len(writer.plans))
	}
}

func TestStartRegistersOwnershipBeforeComposingADSBFlow(t *testing.T) {
	ctx := context.Background()
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.ADSB.Enabled = true
	cfg.ADSB.Platform = "edge-alpha"
	cfg.ADSB.Source = "adsb:opensky:test"
	cfg.ADSB.WriteTimeout = 750 * time.Millisecond
	cfg.ADSB.HTTP.URL = "https://example.test/opensky"
	cfg.ADSB.HTTP.PollInterval = time.Hour
	cfg.ADSB.HTTP.StaleAfter = 3 * time.Hour
	cfg.ADSB.HTTP.ContactPolicy = "semops-test@example.invalid"

	var stoppedOwners bool
	var gotWriterCfg stack.ADSBAdapterConfig
	var gotWriterDeps stack.ADSBAdapterDeps

	app, err := start(ctx, cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			_ context.Context,
			gotClient semstreamsClient,
			heartbeat time.Duration,
			owned []cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			if gotClient != client {
				t.Fatal("ownership registration got unexpected client")
			}
			if !client.connected {
				t.Fatal("ownership registration must run after NATS connect")
			}
			if heartbeat != cfg.OwnershipHeartbeatInterval {
				t.Fatalf("heartbeat interval = %s, want %s", heartbeat, cfg.OwnershipHeartbeatInterval)
			}
			if !hasOwnedContract(owned, cop.OwnerADSB, cop.ADSBTrackContract().Name) {
				t.Fatalf("runtime owned contracts did not include ADS-B when enabled: %+v", owned)
			}
			return copownership.BindingResult{
					Incarnation: "lease-123",
					Owners:      []string{cop.OwnerADSB},
					Tokens: map[string]ownership.OwnerToken{
						cop.OwnerADSB: ownership.ExpectedOwnerToken(cop.OwnerADSB, "lease-123"),
					},
				}, func() {
					stoppedOwners = true
				}, nil
		},
		newADSBPlanWriter: func(
			writerCfg stack.ADSBAdapterConfig,
			writerDeps stack.ADSBAdapterDeps,
		) (adsbcomponent.PlanWriter, error) {
			gotWriterCfg = writerCfg
			gotWriterDeps = writerDeps
			return &recordingADSBAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.ADSBHTTPPoller() == nil || app.ADSBDecoder() == nil || app.ADSBProjector() == nil {
		t.Fatal("expected hosted ADS-B poller, decoder, and projector components")
	}
	if got, want := gotWriterCfg.OwnerTokens[cop.OwnerADSB].Wire(), "semops.feed.adsb#lease-123"; got != want {
		t.Fatalf("writer ADS-B owner token = %q, want %q", got, want)
	}
	if gotWriterCfg.Platform != "edge-alpha" || gotWriterCfg.Source != "adsb:opensky:test" {
		t.Fatalf("writer config = %+v", gotWriterCfg)
	}
	if gotWriterCfg.WriteTimeout != 750*time.Millisecond {
		t.Fatalf("writer timeout = %s", gotWriterCfg.WriteTimeout)
	}
	if gotWriterDeps.NATS != client {
		t.Fatal("ADS-B writer must reuse the connected SemStreams client")
	}
	if got := app.ADSBHTTPPoller().DebugStatus().(adsbcomponent.HTTPPollerDebugStatus).StaleAfter; got != 3*time.Hour {
		t.Fatalf("ADS-B poller stale_after = %s, want 3h", got)
	}
	if !client.hasSubscription(adsbcomponent.DefaultDecodedSubject) ||
		!client.hasSubscription(adsbcomponent.DefaultRawSubject) {
		t.Fatalf("ADS-B flow subscriptions = %+v", client.subscriptionSubjects())
	}

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
	if !stoppedOwners {
		t.Fatal("close must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("close must close SemStreams client")
	}
}

func TestStartADSBFlowCapturesProviderReplayWhenConfigured(t *testing.T) {
	now := time.Date(2026, 6, 20, 19, 0, 0, 0, time.UTC)
	records, err := adsbcodec.OpenSkyFixtureRecords(now)
	if err != nil {
		t.Fatalf("adsb fixtures: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(records[0].RawJSON)
	}))
	defer server.Close()

	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.ADSB.Enabled = true
	cfg.ADSB.Source = "adsb:opensky:runtime-provider-fixture"
	cfg.ADSB.ReplayPath = filepath.Join(t.TempDir(), "adsb-provider.jsonl")
	cfg.ADSB.HTTP.URL = server.URL
	cfg.ADSB.HTTP.PollInterval = time.Hour
	writer := &recordingADSBAppPlanWriter{}

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{
				Incarnation: "lease-123",
				Owners:      []string{cop.OwnerADSB},
				Tokens: map[string]ownership.OwnerToken{
					cop.OwnerADSB: ownership.ExpectedOwnerToken(cop.OwnerADSB, "lease-123"),
				},
			}, func() {}, nil
		},
		newADSBPlanWriter: func(
			stack.ADSBAdapterConfig,
			stack.ADSBAdapterDeps,
		) (adsbcomponent.PlanWriter, error) {
			return writer, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.Close(context.Background()); err != nil {
			t.Fatalf("close app: %v", err)
		}
	})

	if err := app.ADSBHTTPPoller().PollOnce(context.Background()); err != nil {
		t.Fatalf("poll ADS-B provider fixture: %v", err)
	}
	loaded, err := adsbcodec.LoadReplay(cfg.ADSB.ReplayPath)
	if err != nil {
		t.Fatalf("load runtime ADS-B replay: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("runtime ADS-B replay records = %d, want 1", len(loaded))
	}
	if loaded[0].Source != "adsb-opensky-runtime-provider-fixture" {
		t.Fatalf("runtime ADS-B replay record source = %q", loaded[0].Source)
	}
	snapshot, err := loaded[0].Snapshot()
	if err != nil {
		t.Fatalf("decode runtime ADS-B replay snapshot: %v", err)
	}
	if len(snapshot.States) != 2 || snapshot.States[0].ICAO24 != "a1b2c3" {
		t.Fatalf("runtime ADS-B replay snapshot = %+v", snapshot)
	}
	if len(writer.plans) != 1 {
		t.Fatalf("ADS-B graph plans = %d, want aircraft projection", len(writer.plans))
	}
}

func TestStartComposesSAPIENTPreflightFlowWithoutOwnershipExpansion(t *testing.T) {
	ctx := context.Background()
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.SAPIENT.Enabled = true
	cfg.SAPIENT.Source = "sapient:http:test"
	cfg.SAPIENT.HTTP.URL = "https://example.test/sapient"
	cfg.SAPIENT.HTTP.PollInterval = time.Hour
	cfg.SAPIENT.HTTP.StaleAfter = 3 * time.Hour
	cfg.SAPIENT.HTTP.ContactPolicy = "semops-test@example.invalid"

	var stoppedOwners bool
	app, err := start(ctx, cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			_ context.Context,
			gotClient semstreamsClient,
			heartbeat time.Duration,
			owned []cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			if gotClient != client {
				t.Fatal("ownership registration got unexpected client")
			}
			if !client.connected {
				t.Fatal("ownership registration must run after NATS connect")
			}
			if heartbeat != cfg.OwnershipHeartbeatInterval {
				t.Fatalf("heartbeat interval = %s, want %s", heartbeat, cfg.OwnershipHeartbeatInterval)
			}
			for _, item := range owned {
				if strings.Contains(item.Owner, "sapient") {
					t.Fatalf("SAPIENT preflight runtime must not register graph ownership: %+v", owned)
				}
			}
			return copownership.BindingResult{
					Incarnation: "lease-123",
				}, func() {
					stoppedOwners = true
				}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.SAPIENTHTTPInput() == nil || app.SAPIENTDecoder() == nil {
		t.Fatal("expected hosted SAPIENT HTTP input and decoder components")
	}
	if got := app.SAPIENTHTTPInput().DebugStatus().(sapientcomponent.HTTPInputDebugStatus).StaleAfter; got != 3*time.Hour {
		t.Fatalf("SAPIENT stale_after = %s, want 3h", got)
	}
	if !client.hasSubscription(sapientcomponent.DefaultRawSubject) {
		t.Fatalf("SAPIENT flow subscriptions = %+v", client.subscriptionSubjects())
	}
	if client.hasSubscription(sapientcomponent.DefaultDecodedSubject) {
		t.Fatalf("SAPIENT preflight runtime should not subscribe decoded graph subject: %+v", client.subscriptionSubjects())
	}

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
	if !stoppedOwners {
		t.Fatal("close must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("close must close SemStreams client")
	}
}

func TestStartRegistersOwnershipBeforeComposingSAPIENTGraphFlow(t *testing.T) {
	ctx := context.Background()
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.SAPIENT.Enabled = true
	cfg.SAPIENT.GraphEnabled = true
	cfg.SAPIENT.Platform = "edge-alpha"
	cfg.SAPIENT.Source = "sapient:http:test"
	cfg.SAPIENT.TraceID = "sapient-runtime-test"
	cfg.SAPIENT.WriteTimeout = 750 * time.Millisecond
	cfg.SAPIENT.HTTP.URL = "https://example.test/sapient"
	cfg.SAPIENT.HTTP.PollInterval = time.Hour
	cfg.SAPIENT.HTTP.StaleAfter = 3 * time.Hour

	var stoppedOwners bool
	var gotWriterTimeout time.Duration
	var gotWriterRetry natsclient.RetryConfig

	app, err := start(ctx, cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			_ context.Context,
			gotClient semstreamsClient,
			heartbeat time.Duration,
			owned []cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			if gotClient != client {
				t.Fatal("ownership registration got unexpected client")
			}
			if !client.connected {
				t.Fatal("ownership registration must run after NATS connect")
			}
			if heartbeat != cfg.OwnershipHeartbeatInterval {
				t.Fatalf("heartbeat interval = %s, want %s", heartbeat, cfg.OwnershipHeartbeatInterval)
			}
			if !hasOwnedContract(owned, cop.OwnerSAPIENT, cop.SAPIENTTrackContract().Name) {
				t.Fatalf("runtime owned contracts did not include SAPIENT graph contract when enabled: %+v", owned)
			}
			return copownership.BindingResult{
					Incarnation: "lease-123",
					Owners:      []string{cop.OwnerSAPIENT},
					Tokens: map[string]ownership.OwnerToken{
						cop.OwnerSAPIENT: ownership.ExpectedOwnerToken(cop.OwnerSAPIENT, "lease-123"),
					},
				}, func() {
					stoppedOwners = true
				}, nil
		},
		newSAPIENTPlanWriter: func(
			timeout time.Duration,
			retry natsclient.RetryConfig,
			requester graphrequest.RetryRequester,
		) (sapientcomponent.PlanWriter, error) {
			gotWriterTimeout = timeout
			gotWriterRetry = retry
			if requester != client {
				t.Fatal("SAPIENT writer must reuse the connected SemStreams client")
			}
			return &recordingSAPIENTAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.SAPIENTHTTPInput() == nil || app.SAPIENTDecoder() == nil || app.SAPIENTProjector() == nil {
		t.Fatal("expected hosted SAPIENT HTTP input, decoder, and projector components")
	}
	if gotWriterTimeout != 750*time.Millisecond {
		t.Fatalf("writer timeout = %s, want 750ms", gotWriterTimeout)
	}
	if gotWriterRetry != cfg.SAPIENT.Retry {
		t.Fatalf("writer retry = %+v, want %+v", gotWriterRetry, cfg.SAPIENT.Retry)
	}
	if !client.hasSubscription(sapientcomponent.DefaultRawSubject) ||
		!client.hasSubscription(sapientcomponent.DefaultDecodedSubject) {
		t.Fatalf("SAPIENT graph flow subscriptions = %+v", client.subscriptionSubjects())
	}

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
	if !stoppedOwners {
		t.Fatal("close must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("close must close SemStreams client")
	}
}

func TestStartRegistersOwnershipBeforeComposingKLVFlow(t *testing.T) {
	ctx := context.Background()
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.KLV.Enabled = true
	cfg.KLV.Platform = "edge-alpha"
	cfg.KLV.Source = "klv:media-ref:test"
	cfg.KLV.WriteTimeout = 750 * time.Millisecond
	cfg.KLV.MediaPath = t.TempDir()
	cfg.KLV.MediaPattern = "*.ts"

	var stoppedOwners bool
	var gotWriterCfg stack.KLVAdapterConfig
	var gotWriterDeps stack.KLVAdapterDeps

	app, err := start(ctx, cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			_ context.Context,
			gotClient semstreamsClient,
			heartbeat time.Duration,
			owned []cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			if gotClient != client {
				t.Fatal("ownership registration got unexpected client")
			}
			if !client.connected {
				t.Fatal("ownership registration must run after NATS connect")
			}
			if heartbeat != cfg.OwnershipHeartbeatInterval {
				t.Fatalf("heartbeat interval = %s, want %s", heartbeat, cfg.OwnershipHeartbeatInterval)
			}
			if !hasOwnedContract(owned, cop.OwnerKLV, cop.KLVSensorFootprintContract().Name) {
				t.Fatalf("runtime owned contracts did not include KLV when enabled: %+v", owned)
			}
			return copownership.BindingResult{
					Incarnation: "lease-123",
					Owners:      []string{cop.OwnerKLV},
					Tokens: map[string]ownership.OwnerToken{
						cop.OwnerKLV: ownership.ExpectedOwnerToken(cop.OwnerKLV, "lease-123"),
					},
				}, func() {
					stoppedOwners = true
				}, nil
		},
		newKLVPlanWriter: func(
			writerCfg stack.KLVAdapterConfig,
			writerDeps stack.KLVAdapterDeps,
		) (klvcomponent.ProjectorPlanWriter, error) {
			gotWriterCfg = writerCfg
			gotWriterDeps = writerDeps
			return &recordingKLVAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.KLVMediaRefInput() == nil || app.KLVDemux() == nil || app.KLVDecoder() == nil || app.KLVProjector() == nil {
		t.Fatal("expected hosted KLV media-ref input, demux, decoder, and projector components")
	}
	if got, want := gotWriterCfg.OwnerTokens[cop.OwnerKLV].Wire(), "semops.feed.klv#lease-123"; got != want {
		t.Fatalf("writer KLV owner token = %q, want %q", got, want)
	}
	if gotWriterCfg.Platform != "edge-alpha" || gotWriterCfg.Source != "klv:media-ref:test" {
		t.Fatalf("writer config = %+v", gotWriterCfg)
	}
	if gotWriterCfg.WriteTimeout != 750*time.Millisecond {
		t.Fatalf("writer timeout = %s", gotWriterCfg.WriteTimeout)
	}
	if gotWriterDeps.NATS != client {
		t.Fatal("KLV writer must reuse the connected SemStreams client")
	}
	if !client.hasSubscription(klvcomponent.DefaultMediaRefSubject) ||
		!client.hasSubscription(klvcomponent.DefaultPacketSubject) ||
		!client.hasSubscription(klvcomponent.DefaultFrameSubject) {
		t.Fatalf("KLV flow subscriptions = %+v", client.subscriptionSubjects())
	}
	if !hasComponentMetricSource(app.ComponentMetricSources(), "semops-input-klv-media-ref", "klv", "media-ref-input") {
		t.Fatalf("missing KLV media-ref metric source: %+v", componentMetricSourceNames(app.ComponentMetricSources()))
	}

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
	if !stoppedOwners {
		t.Fatal("close must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("close must close SemStreams client")
	}
}

func TestStartRegistersOwnershipBeforeComposingWeatherFixtureFlow(t *testing.T) {
	ctx := context.Background()
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.Weather.Enabled = true
	cfg.Weather.Platform = "edge-alpha"
	cfg.Weather.Source = "weather:fixture:test"
	cfg.Weather.WriteTimeout = 750 * time.Millisecond
	cfg.Weather.Freshness = 45 * time.Minute
	cfg.Weather.MaxObservations = 32
	cfg.Weather.FixturePath = filepath.Join("..", "..", "fixtures", "weather", "open-meteo-point.json")

	var stoppedOwners bool
	var gotWriterCfg stack.WeatherAdapterConfig
	var gotWriterDeps stack.WeatherAdapterDeps
	writer := &recordingWeatherAppPlanWriter{}

	app, err := start(ctx, cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			_ context.Context,
			gotClient semstreamsClient,
			heartbeat time.Duration,
			owned []cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			if gotClient != client {
				t.Fatal("ownership registration got unexpected client")
			}
			if !client.connected {
				t.Fatal("ownership registration must run after NATS connect")
			}
			if heartbeat != cfg.OwnershipHeartbeatInterval {
				t.Fatalf("heartbeat interval = %s, want %s", heartbeat, cfg.OwnershipHeartbeatInterval)
			}
			if !hasOwnedContract(owned, cop.OwnerWeather, cop.WeatherObservationContract().Name) {
				t.Fatalf("runtime owned contracts did not include weather when enabled: %+v", owned)
			}
			return copownership.BindingResult{
					Incarnation: "lease-123",
					Owners:      []string{cop.OwnerWeather},
					Tokens: map[string]ownership.OwnerToken{
						cop.OwnerWeather: ownership.ExpectedOwnerToken(cop.OwnerWeather, "lease-123"),
					},
				}, func() {
					stoppedOwners = true
				}, nil
		},
		newWeatherPlanWriter: func(
			writerCfg stack.WeatherAdapterConfig,
			writerDeps stack.WeatherAdapterDeps,
		) (weathercomponent.PlanWriter, error) {
			gotWriterCfg = writerCfg
			gotWriterDeps = writerDeps
			return writer, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.WeatherInput() == nil || app.WeatherDecoder() == nil || app.WeatherProjector() == nil {
		t.Fatal("expected hosted weather fixture input, decoder, and projector components")
	}
	if got, want := gotWriterCfg.OwnerTokens[cop.OwnerWeather].Wire(), "semops.feed.weather#lease-123"; got != want {
		t.Fatalf("writer weather owner token = %q, want %q", got, want)
	}
	if gotWriterCfg.Platform != "edge-alpha" || gotWriterCfg.Source != "weather:fixture:test" {
		t.Fatalf("writer config = %+v", gotWriterCfg)
	}
	if gotWriterCfg.WriteTimeout != 750*time.Millisecond {
		t.Fatalf("writer timeout = %s", gotWriterCfg.WriteTimeout)
	}
	if gotWriterDeps.NATS != client {
		t.Fatal("weather writer must reuse the connected SemStreams client")
	}
	if !client.hasSubscription(weathercomponent.DefaultRawSubject) ||
		!client.hasSubscription(weathercomponent.DefaultDecodedSubject) {
		t.Fatalf("weather flow subscriptions = %+v", client.subscriptionSubjects())
	}
	if got := client.publishedCount(weathercomponent.DefaultRawSubject); got != 1 {
		t.Fatalf("raw weather messages = %d, want fixture input publish", got)
	}
	if got := client.publishedCount(weathercomponent.DefaultDecodedSubject); got != 1 {
		t.Fatalf("decoded weather messages = %d, want decoder publish", got)
	}
	if len(writer.plans) != 1 {
		t.Fatalf("weather graph plans = %d, want one point forecast plan", len(writer.plans))
	}
	if len(writer.plans[0].Mutations) != 16 {
		t.Fatalf("weather graph mutations = %d, want 16 observations", len(writer.plans[0].Mutations))
	}
	if !hasComponentMetricSource(app.ComponentMetricSources(), "semops-input-weather-fixture", "weather", "fixture-input") ||
		!hasComponentMetricSource(app.ComponentMetricSources(), "semops-processor-weather-decode", "weather", "decoder") ||
		!hasComponentMetricSource(app.ComponentMetricSources(), "semops-processor-weather-project", "weather", "projector") {
		t.Fatalf("missing weather metric source: %+v", componentMetricSourceNames(app.ComponentMetricSources()))
	}

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
	if !stoppedOwners {
		t.Fatal("close must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("close must close SemStreams client")
	}
}

func TestStartSAPIENTPreflightCapturesProviderReplayWhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(runtimeSAPIENTTaskAckJSON))
	}))
	defer server.Close()

	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.SAPIENT.Enabled = true
	cfg.SAPIENT.Source = "sapient-fixture"
	cfg.SAPIENT.ReplayPath = filepath.Join(t.TempDir(), "sapient-provider.jsonl")
	cfg.SAPIENT.HTTP.URL = server.URL
	cfg.SAPIENT.HTTP.PollInterval = time.Hour

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.Close(context.Background()); err != nil {
			t.Fatalf("close app: %v", err)
		}
	})

	if err := app.SAPIENTHTTPInput().PollOnce(context.Background()); err != nil {
		t.Fatalf("poll SAPIENT provider fixture: %v", err)
	}
	if got := client.publishedCount(sapientcomponent.DefaultRawSubject); got != 1 {
		t.Fatalf("raw SAPIENT messages = %d, want 1", got)
	}
	if got := client.publishedCount(sapientcomponent.DefaultDecodedSubject); got != 1 {
		t.Fatalf("decoded SAPIENT messages = %d, want 1", got)
	}
	records, err := sapientcodec.LoadReplay(cfg.SAPIENT.ReplayPath)
	if err != nil {
		t.Fatalf("load runtime SAPIENT replay: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("runtime SAPIENT replay records = %d, want 1", len(records))
	}
	if records[0].Source != "sapient-fixture" ||
		records[0].Content != sapientcodec.ContentTaskAck ||
		records[0].NodeID != "a8654cdf-4328-47de-81fa-c495589e30c8" {
		t.Fatalf("runtime SAPIENT replay record = %+v", records[0])
	}
	if _, err := records[0].Message(nil); err != nil {
		t.Fatalf("parse runtime SAPIENT replay payload: %v", err)
	}
}

func TestStartComponentRuntimeOutlivesStartupContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(sapientcodec.TaskAckFixtureJSON())
	}))
	defer server.Close()

	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.SAPIENT.Enabled = true
	cfg.SAPIENT.Source = "sapient-fixture"
	cfg.SAPIENT.HTTP.URL = server.URL
	cfg.SAPIENT.HTTP.PollInterval = 10 * time.Millisecond
	cfg.SAPIENT.HTTP.StaleAfter = time.Second

	startCtx, cancelStartup := context.WithCancel(context.Background())
	app, err := start(startCtx, cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.Close(context.Background()); err != nil {
			t.Fatalf("close app: %v", err)
		}
	})

	cancelStartup()
	pollUntil(t, 500*time.Millisecond, func() bool {
		return client.publishedCount(sapientcomponent.DefaultDecodedSubject) > 0
	})
}

func TestStartCleansUpWhenCAPCompositionFails(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.CAP.Enabled = true
	writerErr := errors.New("bad cap writer")
	var stoppedOwners bool

	_, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {
				stoppedOwners = true
			}, nil
		},
		newCAPPlanWriter: func(
			stack.CAPAdapterConfig,
			stack.CAPAdapterDeps,
		) (capcomponent.PlanWriter, error) {
			return nil, writerErr
		},
	})
	if !errors.Is(err, writerErr) {
		t.Fatalf("error = %v, want cap writer error", err)
	}
	if !stoppedOwners {
		t.Fatal("failed startup must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("failed startup must close SemStreams client")
	}
}

func TestStartCleansUpWhenADSBCompositionFails(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.ADSB.Enabled = true
	writerErr := errors.New("bad adsb writer")
	var stoppedOwners bool

	_, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {
				stoppedOwners = true
			}, nil
		},
		newADSBPlanWriter: func(
			stack.ADSBAdapterConfig,
			stack.ADSBAdapterDeps,
		) (adsbcomponent.PlanWriter, error) {
			return nil, writerErr
		},
	})
	if !errors.Is(err, writerErr) {
		t.Fatalf("error = %v, want adsb writer error", err)
	}
	if !stoppedOwners {
		t.Fatal("failed startup must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("failed startup must close SemStreams client")
	}
}

func TestStartCleansUpWhenSAPIENTCompositionFails(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.SAPIENT.Enabled = true
	cfg.SAPIENT.HTTP.URL = "https://example.test/sapient"
	decoderErr := errors.New("bad sapient decoder")
	var stoppedOwners bool

	_, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {
				stoppedOwners = true
			}, nil
		},
		newSAPIENTDecoder: func(
			sapientcomponent.DecoderConfig,
			sapientcomponent.Bus,
		) (*sapientcomponent.DecoderComponent, error) {
			return nil, decoderErr
		},
	})
	if !errors.Is(err, decoderErr) {
		t.Fatalf("error = %v, want sapient decoder error", err)
	}
	if !stoppedOwners {
		t.Fatal("failed startup must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("failed startup must close SemStreams client")
	}
}

func TestStartCleansUpWhenKLVCompositionFails(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.KLV.Enabled = true
	cfg.KLV.MediaPath = t.TempDir()
	writerErr := errors.New("bad klv writer")
	var stoppedOwners bool

	_, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {
				stoppedOwners = true
			}, nil
		},
		newKLVPlanWriter: func(
			stack.KLVAdapterConfig,
			stack.KLVAdapterDeps,
		) (klvcomponent.ProjectorPlanWriter, error) {
			return nil, writerErr
		},
	})
	if !errors.Is(err, writerErr) {
		t.Fatalf("error = %v, want klv writer error", err)
	}
	if !stoppedOwners {
		t.Fatal("failed startup must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("failed startup must close SemStreams client")
	}
}

func TestStartCleansUpWhenWeatherCompositionFails(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.Weather.Enabled = true
	cfg.Weather.FixturePath = filepath.Join("..", "..", "fixtures", "weather", "open-meteo-point.json")
	writerErr := errors.New("bad weather writer")
	var stoppedOwners bool

	_, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {
				stoppedOwners = true
			}, nil
		},
		newWeatherPlanWriter: func(
			stack.WeatherAdapterConfig,
			stack.WeatherAdapterDeps,
		) (weathercomponent.PlanWriter, error) {
			return nil, writerErr
		},
	})
	if !errors.Is(err, writerErr) {
		t.Fatalf("error = %v, want weather writer error", err)
	}
	if !stoppedOwners {
		t.Fatal("failed startup must stop ownership heartbeat")
	}
	if !client.closed {
		t.Fatal("failed startup must close SemStreams client")
	}
}

func TestStartCanDisableHostedMAVLinkFlow(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	var composed bool

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
		newMAVLinkPlanWriter: func(
			stack.MAVLinkAdapterConfig,
			stack.MAVLinkAdapterDeps,
		) (mavcomponent.PlanWriter, error) {
			composed = true
			return &recordingAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if composed {
		t.Fatal("MAVLink flow should not be composed when disabled")
	}
	if app.MAVLinkDecoder() != nil || app.MAVLinkProjector() != nil || app.MAVLinkInput() != nil {
		t.Fatal("MAVLink components should be nil when disabled")
	}
}

func TestStartCanDisableHostedCoTFlow(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.CoT.Enabled = false
	var composed bool

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
		newCoTPlanWriter: func(
			stack.CoTAdapterConfig,
			stack.CoTAdapterDeps,
		) (cotcomponent.PlanWriter, error) {
			composed = true
			return &recordingCoTAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if composed {
		t.Fatal("CoT flow should not be composed when disabled")
	}
	if app.CoTDecoder() != nil || app.CoTProjector() != nil || app.CoTUDPInput() != nil || app.CoTTCPInput() != nil {
		t.Fatal("CoT components should be nil when disabled")
	}
}

func TestStartCanDisableHostedCAPFlow(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.CAP.Enabled = false
	var composed bool

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
		newCAPPlanWriter: func(
			stack.CAPAdapterConfig,
			stack.CAPAdapterDeps,
		) (capcomponent.PlanWriter, error) {
			composed = true
			return &recordingCAPAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if composed {
		t.Fatal("CAP flow should not be composed when disabled")
	}
	if app.CAPHTTPPoller() != nil || app.CAPDecoder() != nil || app.CAPProjector() != nil {
		t.Fatal("CAP components should be nil when disabled")
	}
}

func TestStartCanDisableHostedADSBFlow(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.ADSB.Enabled = false
	var composed bool

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
		newADSBPlanWriter: func(
			stack.ADSBAdapterConfig,
			stack.ADSBAdapterDeps,
		) (adsbcomponent.PlanWriter, error) {
			composed = true
			return &recordingADSBAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if composed {
		t.Fatal("ADS-B flow should not be composed when disabled")
	}
	if app.ADSBHTTPPoller() != nil || app.ADSBDecoder() != nil || app.ADSBProjector() != nil {
		t.Fatal("ADS-B components should be nil when disabled")
	}
}

func TestStartCanDisableHostedSAPIENTPreflightFlow(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.SAPIENT.Enabled = false
	var composed bool

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
		newSAPIENTHTTPInput: func(
			sapientcomponent.HTTPInputConfig,
			sapientcomponent.Bus,
		) (*sapientcomponent.HTTPInputComponent, error) {
			composed = true
			return nil, errors.New("should not compose SAPIENT input when disabled")
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if composed {
		t.Fatal("SAPIENT flow should not be composed when disabled")
	}
	if app.SAPIENTHTTPInput() != nil || app.SAPIENTDecoder() != nil {
		t.Fatal("SAPIENT components should be nil when disabled")
	}
}

func TestStartCanDisableHostedWeatherFixtureFlow(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.Weather.Enabled = false
	var composed bool

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
		newWeatherPlanWriter: func(
			stack.WeatherAdapterConfig,
			stack.WeatherAdapterDeps,
		) (weathercomponent.PlanWriter, error) {
			composed = true
			return &recordingWeatherAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if composed {
		t.Fatal("weather flow should not be composed when disabled")
	}
	if app.WeatherInput() != nil || app.WeatherDecoder() != nil || app.WeatherProjector() != nil {
		t.Fatal("weather components should be nil when disabled")
	}
}

func TestStartCanDisableHostedFusionFlow(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.Fusion.Enabled = false
	var composed bool

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
		newFusionPlanWriter: func(
			time.Duration,
			natsclient.RetryConfig,
			graphrequest.RetryRequester,
		) (fusioncomponent.PlanWriter, error) {
			composed = true
			return &recordingFusionAppPlanWriter{}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if composed {
		t.Fatal("fusion flow should not be composed when disabled")
	}
	if app.FusionProjector() != nil {
		t.Fatal("fusion component should be nil when disabled")
	}
}

func TestStartHostsFusionProjectorWhenEnabled(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.Fusion.Enabled = true
	cfg.Fusion.CandidateSubject = "semops.fusion.test_candidates"
	cfg.Fusion.WriteTimeout = 812 * time.Millisecond
	cfg.Fusion.AssociationMaxDistanceMeters = 140
	cfg.Fusion.AssociationMaxTimeDelta = 6 * time.Second
	cfg.Fusion.AssociationMaxObservationAge = 21 * time.Second
	cfg.Fusion.AssociationSourcePriority = []string{"tak", "mavlink"}
	cfg.Fusion.AssociationMinConfidence = 0.7
	cfg.Fusion.AssociationAmbiguityMargin = 0.08
	cfg.Fusion.Retry = natsclient.RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    2 * time.Millisecond,
		MaxBackoff:        20 * time.Millisecond,
		BackoffMultiplier: 2,
	}

	var gotTimeout time.Duration
	var gotRetry natsclient.RetryConfig
	var gotAssociation fusionassociation.Config
	writer := &recordingFusionAppPlanWriter{}

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{
				Incarnation: "lease-123",
				Tokens: map[string]ownership.OwnerToken{
					cop.OwnerFusion: ownership.ExpectedOwnerToken(cop.OwnerFusion, "lease-123"),
				},
			}, func() {}, nil
		},
		newFusionPlanWriter: func(
			timeout time.Duration,
			retry natsclient.RetryConfig,
			requester graphrequest.RetryRequester,
		) (fusioncomponent.PlanWriter, error) {
			if requester != client {
				t.Fatal("fusion writer received unexpected requester")
			}
			gotTimeout = timeout
			gotRetry = retry
			return writer, nil
		},
		newFusionProjector: func(
			projectorCfg fusioncomponent.ProjectorConfig,
			bus fusioncomponent.Bus,
		) (*fusioncomponent.ProjectorComponent, error) {
			gotAssociation = projectorCfg.Association
			return fusioncomponent.NewProjectorComponent(projectorCfg, bus)
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.FusionProjector() == nil {
		t.Fatal("fusion projector was not composed")
	}
	if !client.hasSubscription(cfg.Fusion.CandidateSubject) {
		t.Fatalf("fusion candidate subscriptions = %+v", client.subscriptionSubjects())
	}
	if gotTimeout != cfg.Fusion.WriteTimeout {
		t.Fatalf("fusion write timeout = %s, want %s", gotTimeout, cfg.Fusion.WriteTimeout)
	}
	if gotRetry != cfg.Fusion.Retry {
		t.Fatalf("fusion retry = %+v, want %+v", gotRetry, cfg.Fusion.Retry)
	}
	if gotAssociation.MaxDistanceMeters != 140 ||
		gotAssociation.MaxTimeDelta != 6*time.Second ||
		gotAssociation.MaxObservationAge != 21*time.Second ||
		gotAssociation.MinConfidence != 0.7 ||
		gotAssociation.AmbiguityMargin != 0.08 {
		t.Fatalf("fusion association config = %+v", gotAssociation)
	}
	if len(gotAssociation.SourcePriority) != 2 ||
		gotAssociation.SourcePriority[0] != "tak" ||
		gotAssociation.SourcePriority[1] != "mavlink" {
		t.Fatalf("fusion source priority = %+v", gotAssociation.SourcePriority)
	}
}

func TestStartHostsFusionCandidateProducerWhenEnabled(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.Fusion.CandidateProducerEnabled = true
	cfg.Fusion.CandidateSubject = "semops.fusion.runtime_candidates"
	cfg.Fusion.CandidateSources = []string{"mavlink", "adsb"}
	cfg.Fusion.CandidatePollInterval = time.Hour
	cfg.Fusion.CandidateQueryTimeout = 711 * time.Millisecond
	cfg.Fusion.CandidateLimitPerSource = 7
	cfg.Fusion.CandidateMaxPairComparisons = 11
	cfg.Fusion.CandidateMaxBatches = 3

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.FusionCandidateProducer() == nil {
		t.Fatal("fusion candidate producer was not composed")
	}
	if app.FusionProjector() != nil {
		t.Fatal("fusion projector should not be composed when only candidate production is enabled")
	}
	outputs := app.FusionCandidateProducer().OutputPorts()
	if got := outputs[1].Config.(component.NATSPort).Subject; got != cfg.Fusion.CandidateSubject {
		t.Fatalf("candidate output subject = %q, want %q", got, cfg.Fusion.CandidateSubject)
	}
	if !hasComponentMetricSource(app.ComponentMetricSources(), "semops-processor-fusion-candidates", "fusion", "candidate-producer") {
		t.Fatalf("missing fusion candidate metric source: %+v", componentMetricSourceNames(app.ComponentMetricSources()))
	}
	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
}

func TestStartHostsMAVLinkUDPInputFlowWhenConfigured(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.UDP.ListenAddr = "127.0.0.1:0"
	cfg.MAVLink.UDP.MaxDatagramBytes = 2048

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.MAVLinkInput() == nil {
		t.Fatal("expected hosted MAVLink UDP input component")
	}
	if app.MAVLinkInput().Addr() == nil {
		t.Fatal("expected hosted MAVLink UDP input address")
	}
	if !client.hasSubscription(mavcomponent.DefaultRawSubject) ||
		!client.hasSubscription(mavcomponent.DefaultDecodedSubject) {
		t.Fatalf("subscriptions = %+v", client.subscriptionSubjects())
	}

	conn, err := net.Dial("udp", app.MAVLinkInput().Addr().String())
	if err != nil {
		t.Fatalf("dial MAVLink UDP input: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write(mustRuntimeHeartbeatFrame(t)); err != nil {
		t.Fatalf("write MAVLink UDP frame: %v", err)
	}
	waitFor(t, time.Second, func() bool { return client.requestCount() >= 2 })

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
	if !client.closed {
		t.Fatal("close must close SemStreams client")
	}
}

func TestStartHostsCoTInputFlowWhenConfigured(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.CoT.Enabled = true
	cfg.CoT.UDP.ListenAddr = "127.0.0.1:0"
	cfg.CoT.UDP.MaxDatagramBytes = 2048
	cfg.CoT.TCP.ListenAddr = "127.0.0.1:0"
	cfg.CoT.TCP.MaxEventBytes = 4096

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.CoTUDPInput() == nil || app.CoTTCPInput() == nil {
		t.Fatal("expected hosted CoT UDP and TCP input components")
	}
	if app.CoTUDPInput().Addr() == nil || app.CoTTCPInput().Addr() == nil {
		t.Fatal("expected hosted CoT input addresses")
	}
	if !client.hasSubscription(cotcomponent.DefaultRawSubject) ||
		!client.hasSubscription(cotcomponent.DefaultDecodedSubject) {
		t.Fatalf("subscriptions = %+v", client.subscriptionSubjects())
	}

	conn, err := net.Dial("udp", app.CoTUDPInput().Addr().String())
	if err != nil {
		t.Fatalf("dial CoT UDP input: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write(mustRuntimeCoTEvent(t)); err != nil {
		t.Fatalf("write CoT event: %v", err)
	}
	waitFor(t, time.Second, func() bool { return client.requestCount() >= 2 })

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
	if !client.closed {
		t.Fatal("close must close SemStreams client")
	}
}

func TestRuntimeOwnedContractsIncludeADSBOnlyWhenEnabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ADSB.Enabled = false
	if hasOwnedContract(runtimeOwnedContracts(cfg), cop.OwnerADSB, cop.ADSBTrackContract().Name) {
		t.Fatal("runtime ownership should not include ADS-B when the hosted ADS-B flow is disabled")
	}

	cfg.ADSB.Enabled = true
	if !hasOwnedContract(runtimeOwnedContracts(cfg), cop.OwnerADSB, cop.ADSBTrackContract().Name) {
		t.Fatal("runtime ownership should include ADS-B when the hosted ADS-B flow is enabled")
	}
}

func TestRuntimeOwnedContractsIncludeKLVOnlyWhenEnabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.KLV.Enabled = false
	if hasOwnedContract(runtimeOwnedContracts(cfg), cop.OwnerKLV, cop.KLVSensorFootprintContract().Name) {
		t.Fatal("runtime ownership should not include KLV when the hosted KLV flow is disabled")
	}

	cfg.KLV.Enabled = true
	if !hasOwnedContract(runtimeOwnedContracts(cfg), cop.OwnerKLV, cop.KLVSensorFootprintContract().Name) {
		t.Fatal("runtime ownership should include KLV when the hosted KLV flow is enabled")
	}
}

func TestRuntimeOwnedContractsIncludeWeatherOnlyWhenEnabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Weather.Enabled = false
	if hasOwnedContract(runtimeOwnedContracts(cfg), cop.OwnerWeather, cop.WeatherObservationContract().Name) {
		t.Fatal("runtime ownership should not include weather when the hosted weather flow is disabled")
	}

	cfg.Weather.Enabled = true
	if !hasOwnedContract(runtimeOwnedContracts(cfg), cop.OwnerWeather, cop.WeatherObservationContract().Name) {
		t.Fatal("runtime ownership should include weather when the hosted weather flow is enabled")
	}
}

func TestRuntimeOwnedContractsIncludeSAPIENTOnlyWhenGraphEnabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SAPIENT.Enabled = false
	cfg.SAPIENT.GraphEnabled = false
	if hasOwnedContract(runtimeOwnedContracts(cfg), cop.OwnerSAPIENT, cop.SAPIENTTrackContract().Name) {
		t.Fatal("runtime ownership should not include SAPIENT when disabled")
	}

	cfg.SAPIENT.Enabled = true
	if hasOwnedContract(runtimeOwnedContracts(cfg), cop.OwnerSAPIENT, cop.SAPIENTTrackContract().Name) {
		t.Fatal("runtime ownership should not include SAPIENT for preflight-only runtime")
	}

	cfg.SAPIENT.GraphEnabled = true
	if !hasOwnedContract(runtimeOwnedContracts(cfg), cop.OwnerSAPIENT, cop.SAPIENTTrackContract().Name) {
		t.Fatal("runtime ownership should include SAPIENT when graph projection is enabled")
	}
}

func TestNewSAPIENTPlanWriterUsesConfiguredRetry(t *testing.T) {
	client := &fakeSemStreamsClient{}
	retry := natsclient.RetryConfig{
		MaxRetries:        7,
		InitialBackoff:    3 * time.Millisecond,
		MaxBackoff:        30 * time.Millisecond,
		BackoffMultiplier: 2,
	}
	writer, err := newSAPIENTPlanWriter(45*time.Millisecond, retry, client)
	if err != nil {
		t.Fatalf("new SAPIENT plan writer: %v", err)
	}
	err = writer.Apply(context.Background(), sapientprojector.Plan{Mutations: []sapientprojector.Mutation{{
		Kind: sapientprojector.MutationCreate,
		Create: graph.CreateEntityWithTriplesRequest{
			Entity: &graph.EntityState{ID: "sapient-track-1"},
		},
	}}})
	if err != nil {
		t.Fatalf("apply SAPIENT plan: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(client.requests))
	}
	if client.requests[0].timeout != 45*time.Millisecond {
		t.Fatalf("timeout = %s, want 45ms", client.requests[0].timeout)
	}
	if client.requests[0].retry != retry {
		t.Fatalf("retry = %+v, want %+v", client.requests[0].retry, retry)
	}
}

func TestNewFusionPlanWriterUsesConfiguredRetry(t *testing.T) {
	client := &fakeSemStreamsClient{}
	retry := natsclient.RetryConfig{
		MaxRetries:        6,
		InitialBackoff:    4 * time.Millisecond,
		MaxBackoff:        40 * time.Millisecond,
		BackoffMultiplier: 2,
	}
	writer, err := newFusionPlanWriter(46*time.Millisecond, retry, client)
	if err != nil {
		t.Fatalf("new fusion plan writer: %v", err)
	}
	err = writer.Apply(context.Background(), fusionprojector.Plan{Mutations: []fusionprojector.Mutation{{
		Kind: fusionprojector.MutationCreate,
		Create: graph.CreateEntityWithTriplesRequest{
			Entity: &graph.EntityState{ID: "fusion-association-1"},
		},
	}}})
	if err != nil {
		t.Fatalf("apply fusion plan: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(client.requests))
	}
	if client.requests[0].timeout != 46*time.Millisecond {
		t.Fatalf("timeout = %s, want 46ms", client.requests[0].timeout)
	}
	if client.requests[0].retry != retry {
		t.Fatalf("retry = %+v, want %+v", client.requests[0].retry, retry)
	}
}

func TestComponentMetricSourcesExposeStartedRuntimeComponents(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	defer app.Close(context.Background())

	sources := app.ComponentMetricSources()
	if !hasComponentMetricSource(sources, "semops-processor-mavlink-decode", "mavlink", "decoder") {
		t.Fatalf("missing MAVLink decoder metric source: %+v", componentMetricSourceNames(sources))
	}
	if !hasComponentMetricSource(sources, "semops-processor-mavlink-project", "mavlink", "projector") {
		t.Fatalf("missing MAVLink projector metric source: %+v", componentMetricSourceNames(sources))
	}
	if hasComponentMetricSource(sources, "semops-input-mavlink-udp", "mavlink", "input") {
		t.Fatalf("unexpected MAVLink UDP source without listen addr: %+v", componentMetricSourceNames(sources))
	}
}

func TestComponentMetricSourcesExposeHostedFusionProjector(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.Fusion.Enabled = true

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
			[]cop.OwnedContract,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{
				Incarnation: "lease-123",
				Tokens: map[string]ownership.OwnerToken{
					cop.OwnerFusion: ownership.ExpectedOwnerToken(cop.OwnerFusion, "lease-123"),
				},
			}, func() {}, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	sources := app.ComponentMetricSources()
	if !hasComponentMetricSource(sources, "semops-processor-fusion-associate", "fusion", "projector") {
		t.Fatalf("missing fusion metric source: %+v", componentMetricSourceNames(sources))
	}
}

func TestConfigFromEnv(t *testing.T) {
	env := map[string]string{
		EnvNATSURL:                       "nats://semstreams:4222",
		EnvNATSName:                      "semops-test",
		EnvNATSConnectTimeout:            "3s",
		EnvAPIAddr:                       ":18088",
		EnvOwnershipHeartbeatInterval:    "4s",
		EnvCOPGraphQueryTimeout:          "1500ms",
		EnvCOPGraphDiscoveryEnabled:      "false",
		EnvCOPGraphDiscoveryLimit:        "75",
		EnvCOPMAVLinkSystemIDs:           "42, 43",
		EnvCOPCoTUIDs:                    "ANDROID-ALPHA, MARKER-NORTH-GATE",
		EnvCOPCAPAlertIDs:                "nws-demo-flood-warning, nws-demo-flood-update",
		EnvCOPOperatorIdentityMode:       COPOperatorIdentityModeTrustedHeaders,
		EnvMAVLinkEnabled:                "false",
		EnvMAVLinkSource:                 "udp:14550",
		EnvOrg:                           "lab",
		EnvPlatform:                      "edge-7",
		EnvTraceID:                       "trace-7",
		EnvMAVLinkWriteTimeout:           "900ms",
		EnvMAVLinkUDPListenAddr:          "127.0.0.1:14550",
		EnvMAVLinkUDPMaxDatagramBytes:    "2048",
		EnvCoTEnabled:                    "true",
		EnvCoTSource:                     "udp:cot",
		EnvCoTWriteTimeout:               "950ms",
		EnvCoTUDPListenAddr:              "127.0.0.1:18080",
		EnvCoTUDPMaxDatagramBytes:        "4096",
		EnvCoTTCPListenAddr:              "127.0.0.1:18081",
		EnvCoTTCPMaxEventBytes:           "8192",
		EnvCAPEnabled:                    "true",
		EnvCAPSource:                     "cap:http",
		EnvCAPReplayPath:                 "/tmp/semops-cap.jsonl",
		EnvCAPWriteTimeout:               "975ms",
		EnvCAPHTTPURL:                    "https://example.test/cap",
		EnvCAPHTTPMethod:                 "POST",
		EnvCAPHTTPPollInterval:           "45s",
		EnvCAPHTTPStaleAfter:             "2m",
		EnvCAPHTTPContactPolicy:          "semops-test@example.invalid",
		EnvCAPHTTPAuthRef:                "cap-secret",
		EnvCAPHTTPMaxResponseBytes:       "123456",
		EnvADSBEnabled:                   "true",
		EnvADSBSource:                    "adsb:opensky",
		EnvADSBReplayPath:                "/tmp/semops-adsb.jsonl",
		EnvADSBRawMaxRecords:             "32",
		EnvADSBRawMaxBytes:               "65536",
		EnvADSBWriteTimeout:              "980ms",
		EnvADSBHTTPURL:                   "https://example.test/opensky",
		EnvADSBHTTPMethod:                "POST",
		EnvADSBHTTPPollInterval:          "40s",
		EnvADSBHTTPStaleAfter:            "3m",
		EnvADSBHTTPContactPolicy:         "semops-adsb-test@example.invalid",
		EnvADSBHTTPAuthRef:               "adsb-secret",
		EnvADSBHTTPMaxResponseBytes:      "234567",
		EnvSAPIENTEnabled:                "true",
		EnvSAPIENTGraphEnabled:           "true",
		EnvSAPIENTSource:                 "sapient:http",
		EnvSAPIENTReplayPath:             "/tmp/semops-sapient.jsonl",
		EnvSAPIENTRawMaxRecords:          "33",
		EnvSAPIENTRawMaxBytes:            "75536",
		EnvSAPIENTWriteTimeout:           "985ms",
		EnvSAPIENTHTTPURL:                "https://example.test/sapient",
		EnvSAPIENTHTTPMethod:             "POST",
		EnvSAPIENTHTTPPollInterval:       "50s",
		EnvSAPIENTHTTPStaleAfter:         "4m",
		EnvSAPIENTHTTPContactPolicy:      "semops-sapient-test@example.invalid",
		EnvSAPIENTHTTPAuthRef:            "sapient-secret",
		EnvSAPIENTHTTPMaxResponseBytes:   "345678",
		EnvSAPIENTHTTPEncoding:           "json",
		EnvKLVEnabled:                    "true",
		EnvKLVSource:                     "klv:fixture",
		EnvKLVMediaPath:                  "/tmp/semops-klv",
		EnvKLVMediaPattern:               "*.mpg",
		EnvKLVWriteTimeout:               "990ms",
		EnvKLVDemuxMaxPacketBytes:        "65536",
		EnvKLVDemuxMaxExtractBytes:       "262144",
		EnvKLVDemuxMaxPackets:            "17",
		EnvKLVDemuxMaxMaterializedBytes:  "1048576",
		EnvKLVDemuxProbeOutputMaxBytes:   "32768",
		EnvKLVDecodeMaxPacketBytes:       "65536",
		EnvWeatherEnabled:                "true",
		EnvWeatherSource:                 "weather:fixture",
		EnvWeatherProvider:               weathercodec.ProviderOpenMeteo,
		EnvWeatherQueryShape:             weathercodec.QueryShapePosition,
		EnvWeatherFixturePath:            "/tmp/weather/open-meteo-point.json",
		EnvWeatherWriteTimeout:           "995ms",
		EnvWeatherFreshness:              "45m",
		EnvWeatherMaxObservations:        "48",
		EnvFusionEnabled:                 "true",
		EnvFusionCandidateSubject:        "semops.fusion.env_candidates",
		EnvFusionWriteTimeout:            "996ms",
		EnvFusionCandidatesEnabled:       "true",
		EnvFusionCandidateSources:        "mavlink, adsb",
		EnvFusionCandidatePollInterval:   "12s",
		EnvFusionCandidateQueryTimeout:   "997ms",
		EnvFusionCandidateLimitPerSource: "9",
		EnvFusionCandidateMaxComparisons: "19",
		EnvFusionCandidateMaxBatches:     "4",
	}
	env[EnvFusionAssociationMaxDistance] = "123.5"
	env[EnvFusionAssociationMaxTimeDelta] = "8s"
	env[EnvFusionAssociationMaxObservationAge] = "45s"
	env[EnvFusionAssociationSourcePriority] = "tak, mavlink, adsb"
	env[EnvFusionAssociationMinConfidence] = "0.72"
	env[EnvFusionAssociationAmbiguityMargin] = "0.12"

	cfg, err := ConfigFromEnv(func(name string) string { return env[name] })
	if err != nil {
		t.Fatalf("config from env: %v", err)
	}
	if cfg.NATSURL != "nats://semstreams:4222" || cfg.NATSName != "semops-test" {
		t.Fatalf("NATS config = %+v", cfg)
	}
	if cfg.NATSConnectTimeout != 3*time.Second {
		t.Fatalf("connect timeout = %s", cfg.NATSConnectTimeout)
	}
	if cfg.APIAddr != ":18088" {
		t.Fatalf("api addr = %q", cfg.APIAddr)
	}
	if cfg.OwnershipHeartbeatInterval != 4*time.Second {
		t.Fatalf("heartbeat interval = %s", cfg.OwnershipHeartbeatInterval)
	}
	if cfg.COP.GraphQueryTimeout != 1500*time.Millisecond {
		t.Fatalf("COP graph query timeout = %s", cfg.COP.GraphQueryTimeout)
	}
	if cfg.COP.GraphDiscoveryEnabled {
		t.Fatal("COP graph discovery enabled = true, want false")
	}
	if cfg.COP.GraphDiscoveryLimit != 75 {
		t.Fatalf("COP graph discovery limit = %d", cfg.COP.GraphDiscoveryLimit)
	}
	if len(cfg.COP.MAVLinkSystemIDs) != 2 || cfg.COP.MAVLinkSystemIDs[0] != 42 || cfg.COP.MAVLinkSystemIDs[1] != 43 {
		t.Fatalf("COP MAVLink systems = %+v", cfg.COP.MAVLinkSystemIDs)
	}
	if len(cfg.COP.CoTUIDs) != 2 || cfg.COP.CoTUIDs[0] != "ANDROID-ALPHA" || cfg.COP.CoTUIDs[1] != "MARKER-NORTH-GATE" {
		t.Fatalf("COP CoT UIDs = %+v", cfg.COP.CoTUIDs)
	}
	if len(cfg.COP.CAPAlertIDs) != 2 ||
		cfg.COP.CAPAlertIDs[0] != "nws-demo-flood-warning" ||
		cfg.COP.CAPAlertIDs[1] != "nws-demo-flood-update" {
		t.Fatalf("COP CAP alert IDs = %+v", cfg.COP.CAPAlertIDs)
	}
	if cfg.COP.OperatorIdentityMode != COPOperatorIdentityModeTrustedHeaders {
		t.Fatalf("COP operator identity mode = %q", cfg.COP.OperatorIdentityMode)
	}
	if cfg.MAVLink.Enabled {
		t.Fatal("MAVLink enabled = true, want false")
	}
	if cfg.MAVLink.Source != "udp:14550" ||
		cfg.MAVLink.Org != "lab" ||
		cfg.MAVLink.Platform != "edge-7" ||
		cfg.MAVLink.TraceID != "trace-7" {
		t.Fatalf("MAVLink config = %+v", cfg.MAVLink)
	}
	if cfg.MAVLink.WriteTimeout != 900*time.Millisecond {
		t.Fatalf("MAVLink write timeout = %s", cfg.MAVLink.WriteTimeout)
	}
	if cfg.MAVLink.UDP.ListenAddr != "127.0.0.1:14550" ||
		cfg.MAVLink.UDP.MaxDatagramBytes != 2048 {
		t.Fatalf("MAVLink UDP config = %+v", cfg.MAVLink.UDP)
	}
	if !cfg.CoT.Enabled {
		t.Fatal("CoT enabled = false, want true")
	}
	if cfg.CoT.Source != "udp:cot" ||
		cfg.CoT.Org != "lab" ||
		cfg.CoT.Platform != "edge-7" ||
		cfg.CoT.TraceID != "trace-7" {
		t.Fatalf("CoT config = %+v", cfg.CoT)
	}
	if cfg.CoT.WriteTimeout != 950*time.Millisecond {
		t.Fatalf("CoT write timeout = %s", cfg.CoT.WriteTimeout)
	}
	if cfg.CoT.UDP.ListenAddr != "127.0.0.1:18080" ||
		cfg.CoT.UDP.MaxDatagramBytes != 4096 {
		t.Fatalf("CoT UDP config = %+v", cfg.CoT.UDP)
	}
	if cfg.CoT.TCP.ListenAddr != "127.0.0.1:18081" ||
		cfg.CoT.TCP.MaxEventBytes != 8192 {
		t.Fatalf("CoT TCP config = %+v", cfg.CoT.TCP)
	}
	if !cfg.CAP.Enabled {
		t.Fatal("CAP enabled = false, want true")
	}
	if cfg.CAP.Source != "cap:http" ||
		cfg.CAP.Org != "lab" ||
		cfg.CAP.Platform != "edge-7" ||
		cfg.CAP.TraceID != "trace-7" ||
		cfg.CAP.ReplayPath != "/tmp/semops-cap.jsonl" {
		t.Fatalf("CAP config = %+v", cfg.CAP)
	}
	if cfg.CAP.WriteTimeout != 975*time.Millisecond {
		t.Fatalf("CAP write timeout = %s", cfg.CAP.WriteTimeout)
	}
	if cfg.CAP.HTTP.URL != "https://example.test/cap" ||
		cfg.CAP.HTTP.Method != "POST" ||
		cfg.CAP.HTTP.PollInterval != 45*time.Second ||
		cfg.CAP.HTTP.StaleAfter != 2*time.Minute ||
		cfg.CAP.HTTP.ContactPolicy != "semops-test@example.invalid" ||
		cfg.CAP.HTTP.AuthRef != "cap-secret" ||
		cfg.CAP.HTTP.MaxResponseBytes != 123456 {
		t.Fatalf("CAP HTTP config = %+v", cfg.CAP.HTTP)
	}
	if !cfg.ADSB.Enabled {
		t.Fatal("ADS-B enabled = false, want true")
	}
	if cfg.ADSB.Source != "adsb:opensky" ||
		cfg.ADSB.Org != "lab" ||
		cfg.ADSB.Platform != "edge-7" ||
		cfg.ADSB.TraceID != "trace-7" ||
		cfg.ADSB.ReplayPath != "/tmp/semops-adsb.jsonl" {
		t.Fatalf("ADS-B config = %+v", cfg.ADSB)
	}
	if cfg.ADSB.RawMaxRecords != 32 || cfg.ADSB.RawMaxBytes != 65536 {
		t.Fatalf("ADS-B raw lane config = records %d bytes %d", cfg.ADSB.RawMaxRecords, cfg.ADSB.RawMaxBytes)
	}
	if cfg.ADSB.WriteTimeout != 980*time.Millisecond {
		t.Fatalf("ADS-B write timeout = %s", cfg.ADSB.WriteTimeout)
	}
	if cfg.ADSB.HTTP.URL != "https://example.test/opensky" ||
		cfg.ADSB.HTTP.Method != "POST" ||
		cfg.ADSB.HTTP.PollInterval != 40*time.Second ||
		cfg.ADSB.HTTP.StaleAfter != 3*time.Minute ||
		cfg.ADSB.HTTP.ContactPolicy != "semops-adsb-test@example.invalid" ||
		cfg.ADSB.HTTP.AuthRef != "adsb-secret" ||
		cfg.ADSB.HTTP.MaxResponseBytes != 234567 {
		t.Fatalf("ADS-B HTTP config = %+v", cfg.ADSB.HTTP)
	}
	if !cfg.SAPIENT.Enabled {
		t.Fatal("SAPIENT enabled = false, want true")
	}
	if !cfg.SAPIENT.GraphEnabled {
		t.Fatal("SAPIENT graph enabled = false, want true")
	}
	if cfg.SAPIENT.Source != "sapient:http" ||
		cfg.SAPIENT.Org != "lab" ||
		cfg.SAPIENT.Platform != "edge-7" ||
		cfg.SAPIENT.TraceID != "trace-7" ||
		cfg.SAPIENT.ReplayPath != "/tmp/semops-sapient.jsonl" {
		t.Fatalf("SAPIENT config = %+v", cfg.SAPIENT)
	}
	if cfg.SAPIENT.RawMaxRecords != 33 || cfg.SAPIENT.RawMaxBytes != 75536 {
		t.Fatalf("SAPIENT raw lane config = records %d bytes %d", cfg.SAPIENT.RawMaxRecords, cfg.SAPIENT.RawMaxBytes)
	}
	if cfg.SAPIENT.WriteTimeout != 985*time.Millisecond {
		t.Fatalf("SAPIENT write timeout = %s", cfg.SAPIENT.WriteTimeout)
	}
	if cfg.SAPIENT.HTTP.URL != "https://example.test/sapient" ||
		cfg.SAPIENT.HTTP.Method != "POST" ||
		cfg.SAPIENT.HTTP.PollInterval != 50*time.Second ||
		cfg.SAPIENT.HTTP.StaleAfter != 4*time.Minute ||
		cfg.SAPIENT.HTTP.ContactPolicy != "semops-sapient-test@example.invalid" ||
		cfg.SAPIENT.HTTP.AuthRef != "sapient-secret" ||
		cfg.SAPIENT.HTTP.MaxResponseBytes != 345678 ||
		cfg.SAPIENT.HTTP.Encoding != sapientcodec.EncodingJSON {
		t.Fatalf("SAPIENT HTTP config = %+v", cfg.SAPIENT.HTTP)
	}
	if !cfg.KLV.Enabled {
		t.Fatal("KLV enabled = false, want true")
	}
	if cfg.KLV.Source != "klv:fixture" ||
		cfg.KLV.Org != "lab" ||
		cfg.KLV.Platform != "edge-7" ||
		cfg.KLV.TraceID != "trace-7" ||
		cfg.KLV.MediaPath != "/tmp/semops-klv" ||
		cfg.KLV.MediaPattern != "*.mpg" {
		t.Fatalf("KLV config = %+v", cfg.KLV)
	}
	if cfg.KLV.WriteTimeout != 990*time.Millisecond {
		t.Fatalf("KLV write timeout = %s", cfg.KLV.WriteTimeout)
	}
	if cfg.KLV.Demux.MaxPacketBytes != 65536 ||
		cfg.KLV.Demux.MaxExtractBytes != 262144 ||
		cfg.KLV.Demux.MaxPackets != 17 ||
		cfg.KLV.Demux.MaxMaterializedBytes != 1048576 ||
		cfg.KLV.Demux.ProbeOutputMaxBytes != 32768 {
		t.Fatalf("KLV demux config = %+v", cfg.KLV.Demux)
	}
	if cfg.KLV.Decode.MaxPacketBytes != 65536 {
		t.Fatalf("KLV decode config = %+v", cfg.KLV.Decode)
	}
	if !cfg.Weather.Enabled {
		t.Fatal("weather enabled = false, want true")
	}
	if cfg.Weather.Source != "weather:fixture" ||
		cfg.Weather.Org != "lab" ||
		cfg.Weather.Platform != "edge-7" ||
		cfg.Weather.TraceID != "trace-7" ||
		cfg.Weather.Provider != weathercodec.ProviderOpenMeteo ||
		cfg.Weather.QueryShape != weathercodec.QueryShapePosition ||
		cfg.Weather.FixturePath != "/tmp/weather/open-meteo-point.json" {
		t.Fatalf("weather config = %+v", cfg.Weather)
	}
	if cfg.Weather.WriteTimeout != 995*time.Millisecond {
		t.Fatalf("weather write timeout = %s", cfg.Weather.WriteTimeout)
	}
	if cfg.Weather.Freshness != 45*time.Minute {
		t.Fatalf("weather freshness = %s", cfg.Weather.Freshness)
	}
	if cfg.Weather.MaxObservations != 48 {
		t.Fatalf("weather max observations = %d", cfg.Weather.MaxObservations)
	}
	if !cfg.Fusion.Enabled {
		t.Fatal("fusion enabled = false, want true")
	}
	if !cfg.Fusion.CandidateProducerEnabled {
		t.Fatal("fusion candidate producer enabled = false, want true")
	}
	if cfg.Fusion.Org != "lab" ||
		cfg.Fusion.Platform != "edge-7" ||
		cfg.Fusion.TraceID != "trace-7" ||
		cfg.Fusion.CandidateSubject != "semops.fusion.env_candidates" {
		t.Fatalf("fusion config = %+v", cfg.Fusion)
	}
	if len(cfg.Fusion.CandidateSources) != 2 ||
		cfg.Fusion.CandidateSources[0] != "mavlink" ||
		cfg.Fusion.CandidateSources[1] != "adsb" {
		t.Fatalf("fusion candidate sources = %+v", cfg.Fusion.CandidateSources)
	}
	if cfg.Fusion.CandidatePollInterval != 12*time.Second ||
		cfg.Fusion.CandidateQueryTimeout != 997*time.Millisecond ||
		cfg.Fusion.CandidateLimitPerSource != 9 ||
		cfg.Fusion.CandidateMaxPairComparisons != 19 ||
		cfg.Fusion.CandidateMaxBatches != 4 {
		t.Fatalf("fusion candidate config = %+v", cfg.Fusion)
	}
	if cfg.Fusion.AssociationMaxDistanceMeters != 123.5 ||
		cfg.Fusion.AssociationMaxTimeDelta != 8*time.Second ||
		cfg.Fusion.AssociationMaxObservationAge != 45*time.Second ||
		cfg.Fusion.AssociationMinConfidence != 0.72 ||
		cfg.Fusion.AssociationAmbiguityMargin != 0.12 {
		t.Fatalf("fusion association config = %+v", cfg.Fusion)
	}
	if len(cfg.Fusion.AssociationSourcePriority) != 3 ||
		cfg.Fusion.AssociationSourcePriority[0] != "tak" ||
		cfg.Fusion.AssociationSourcePriority[1] != "mavlink" ||
		cfg.Fusion.AssociationSourcePriority[2] != "adsb" {
		t.Fatalf("fusion source priority = %+v", cfg.Fusion.AssociationSourcePriority)
	}
	if cfg.Fusion.WriteTimeout != 996*time.Millisecond {
		t.Fatalf("fusion write timeout = %s", cfg.Fusion.WriteTimeout)
	}
}

func TestConfigDefaultsUseDiscoveryForCoTCAPSnapshotState(t *testing.T) {
	cfg, err := ConfigFromEnv(func(string) string { return "" })
	if err != nil {
		t.Fatalf("config from defaults: %v", err)
	}
	if !cfg.COP.GraphDiscoveryEnabled {
		t.Fatal("COP graph discovery enabled = false, want true")
	}
	if len(cfg.COP.CoTUIDs) != 0 {
		t.Fatalf("default CoT UIDs = %+v, want discovery-only path", cfg.COP.CoTUIDs)
	}
	if len(cfg.COP.CAPAlertIDs) != 0 {
		t.Fatalf("default CAP alert IDs = %+v, want discovery-only path", cfg.COP.CAPAlertIDs)
	}
}

func TestConfigFromEnvReportsBadValues(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "bad write timeout",
			env:  map[string]string{EnvMAVLinkWriteTimeout: "forever"},
			want: EnvMAVLinkWriteTimeout,
		},
		{
			name: "bad graph query timeout",
			env:  map[string]string{EnvCOPGraphQueryTimeout: "forever"},
			want: EnvCOPGraphQueryTimeout,
		},
		{
			name: "bad graph discovery enabled",
			env:  map[string]string{EnvCOPGraphDiscoveryEnabled: "sometimes"},
			want: EnvCOPGraphDiscoveryEnabled,
		},
		{
			name: "bad graph discovery limit",
			env:  map[string]string{EnvCOPGraphDiscoveryLimit: "many"},
			want: EnvCOPGraphDiscoveryLimit,
		},
		{
			name: "zero graph discovery limit",
			env:  map[string]string{EnvCOPGraphDiscoveryLimit: "0"},
			want: EnvCOPGraphDiscoveryLimit,
		},
		{
			name: "bad mavlink system ids",
			env:  map[string]string{EnvCOPMAVLinkSystemIDs: "42,bad"},
			want: EnvCOPMAVLinkSystemIDs,
		},
		{
			name: "empty mavlink system ids",
			env:  map[string]string{EnvCOPMAVLinkSystemIDs: ","},
			want: EnvCOPMAVLinkSystemIDs,
		},
		{
			name: "out of range mavlink system ids",
			env:  map[string]string{EnvCOPMAVLinkSystemIDs: "300"},
			want: EnvCOPMAVLinkSystemIDs,
		},
		{
			name: "empty cot uids",
			env:  map[string]string{EnvCOPCoTUIDs: ","},
			want: EnvCOPCoTUIDs,
		},
		{
			name: "discovery disabled without cot uids",
			env: map[string]string{
				EnvCOPGraphDiscoveryEnabled: "false",
				EnvCOPCoTUIDs:               "",
				EnvCOPCAPAlertIDs:           "nws-demo-flood-warning",
			},
			want: EnvCOPCoTUIDs,
		},
		{
			name: "empty cap alert ids",
			env:  map[string]string{EnvCOPCAPAlertIDs: ","},
			want: EnvCOPCAPAlertIDs,
		},
		{
			name: "discovery disabled without cap alert ids",
			env: map[string]string{
				EnvCOPGraphDiscoveryEnabled: "false",
				EnvCOPCoTUIDs:               "ANDROID-ALPHA",
				EnvCOPCAPAlertIDs:           "",
			},
			want: EnvCOPCAPAlertIDs,
		},
		{
			name: "bad operator identity mode",
			env:  map[string]string{EnvCOPOperatorIdentityMode: "trust_me"},
			want: EnvCOPOperatorIdentityMode,
		},
		{
			name: "bad udp max datagram",
			env:  map[string]string{EnvMAVLinkUDPMaxDatagramBytes: "huge"},
			want: EnvMAVLinkUDPMaxDatagramBytes,
		},
		{
			name: "bad cot write timeout",
			env:  map[string]string{EnvCoTWriteTimeout: "forever"},
			want: EnvCoTWriteTimeout,
		},
		{
			name: "bad cot udp max datagram",
			env:  map[string]string{EnvCoTUDPMaxDatagramBytes: "huge"},
			want: EnvCoTUDPMaxDatagramBytes,
		},
		{
			name: "bad cot tcp max event bytes",
			env:  map[string]string{EnvCoTTCPMaxEventBytes: "huge"},
			want: EnvCoTTCPMaxEventBytes,
		},
		{
			name: "bad cap enabled",
			env:  map[string]string{EnvCAPEnabled: "sometimes"},
			want: EnvCAPEnabled,
		},
		{
			name: "bad cap write timeout",
			env:  map[string]string{EnvCAPWriteTimeout: "forever"},
			want: EnvCAPWriteTimeout,
		},
		{
			name: "bad cap poll interval",
			env:  map[string]string{EnvCAPHTTPPollInterval: "soon"},
			want: EnvCAPHTTPPollInterval,
		},
		{
			name: "bad cap stale after",
			env:  map[string]string{EnvCAPHTTPStaleAfter: "later"},
			want: EnvCAPHTTPStaleAfter,
		},
		{
			name: "bad cap max response bytes",
			env:  map[string]string{EnvCAPHTTPMaxResponseBytes: "huge"},
			want: EnvCAPHTTPMaxResponseBytes,
		},
		{
			name: "bad adsb enabled",
			env:  map[string]string{EnvADSBEnabled: "sometimes"},
			want: EnvADSBEnabled,
		},
		{
			name: "bad adsb write timeout",
			env:  map[string]string{EnvADSBWriteTimeout: "forever"},
			want: EnvADSBWriteTimeout,
		},
		{
			name: "bad adsb poll interval",
			env:  map[string]string{EnvADSBHTTPPollInterval: "soon"},
			want: EnvADSBHTTPPollInterval,
		},
		{
			name: "bad adsb stale after",
			env:  map[string]string{EnvADSBHTTPStaleAfter: "later"},
			want: EnvADSBHTTPStaleAfter,
		},
		{
			name: "bad adsb raw max records",
			env:  map[string]string{EnvADSBRawMaxRecords: "many"},
			want: EnvADSBRawMaxRecords,
		},
		{
			name: "bad adsb raw max bytes",
			env:  map[string]string{EnvADSBRawMaxBytes: "huge"},
			want: EnvADSBRawMaxBytes,
		},
		{
			name: "bad adsb max response bytes",
			env:  map[string]string{EnvADSBHTTPMaxResponseBytes: "huge"},
			want: EnvADSBHTTPMaxResponseBytes,
		},
		{
			name: "bad sapient enabled",
			env:  map[string]string{EnvSAPIENTEnabled: "sometimes"},
			want: EnvSAPIENTEnabled,
		},
		{
			name: "bad sapient graph enabled",
			env:  map[string]string{EnvSAPIENTGraphEnabled: "sometimes"},
			want: EnvSAPIENTGraphEnabled,
		},
		{
			name: "bad sapient write timeout",
			env:  map[string]string{EnvSAPIENTWriteTimeout: "forever"},
			want: EnvSAPIENTWriteTimeout,
		},
		{
			name: "bad sapient poll interval",
			env:  map[string]string{EnvSAPIENTHTTPPollInterval: "soon"},
			want: EnvSAPIENTHTTPPollInterval,
		},
		{
			name: "bad sapient stale after",
			env:  map[string]string{EnvSAPIENTHTTPStaleAfter: "later"},
			want: EnvSAPIENTHTTPStaleAfter,
		},
		{
			name: "bad sapient raw max records",
			env:  map[string]string{EnvSAPIENTRawMaxRecords: "many"},
			want: EnvSAPIENTRawMaxRecords,
		},
		{
			name: "bad sapient raw max bytes",
			env:  map[string]string{EnvSAPIENTRawMaxBytes: "huge"},
			want: EnvSAPIENTRawMaxBytes,
		},
		{
			name: "bad sapient max response bytes",
			env:  map[string]string{EnvSAPIENTHTTPMaxResponseBytes: "huge"},
			want: EnvSAPIENTHTTPMaxResponseBytes,
		},
		{
			name: "zero udp max datagram",
			env: map[string]string{
				EnvMAVLinkUDPListenAddr:       "127.0.0.1:14550",
				EnvMAVLinkUDPMaxDatagramBytes: "0",
			},
			want: EnvMAVLinkUDPMaxDatagramBytes,
		},
		{
			name: "zero cot udp max datagram",
			env: map[string]string{
				EnvCoTEnabled:             "true",
				EnvCoTUDPListenAddr:       "127.0.0.1:18080",
				EnvCoTUDPMaxDatagramBytes: "0",
			},
			want: EnvCoTUDPMaxDatagramBytes,
		},
		{
			name: "zero cot tcp max event bytes",
			env: map[string]string{
				EnvCoTEnabled:          "true",
				EnvCoTTCPListenAddr:    "127.0.0.1:18081",
				EnvCoTTCPMaxEventBytes: "0",
			},
			want: EnvCoTTCPMaxEventBytes,
		},
		{
			name: "zero cap max response bytes",
			env: map[string]string{
				EnvCAPEnabled:              "true",
				EnvCAPHTTPMaxResponseBytes: "0",
			},
			want: EnvCAPHTTPMaxResponseBytes,
		},
		{
			name: "zero cap stale after",
			env: map[string]string{
				EnvCAPEnabled:        "true",
				EnvCAPHTTPStaleAfter: "0s",
			},
			want: EnvCAPHTTPStaleAfter,
		},
		{
			name: "zero adsb raw max records",
			env: map[string]string{
				EnvADSBEnabled:       "true",
				EnvADSBRawMaxRecords: "0",
			},
			want: EnvADSBRawMaxRecords,
		},
		{
			name: "zero adsb raw max bytes",
			env: map[string]string{
				EnvADSBEnabled:     "true",
				EnvADSBRawMaxBytes: "0",
			},
			want: EnvADSBRawMaxBytes,
		},
		{
			name: "zero adsb max response bytes",
			env: map[string]string{
				EnvADSBEnabled:              "true",
				EnvADSBHTTPMaxResponseBytes: "0",
			},
			want: EnvADSBHTTPMaxResponseBytes,
		},
		{
			name: "zero adsb stale after",
			env: map[string]string{
				EnvADSBEnabled:        "true",
				EnvADSBHTTPStaleAfter: "0s",
			},
			want: EnvADSBHTTPStaleAfter,
		},
		{
			name: "zero sapient raw max records",
			env: map[string]string{
				EnvSAPIENTEnabled:       "true",
				EnvSAPIENTHTTPURL:       "https://example.test/sapient",
				EnvSAPIENTRawMaxRecords: "0",
			},
			want: EnvSAPIENTRawMaxRecords,
		},
		{
			name: "zero sapient raw max bytes",
			env: map[string]string{
				EnvSAPIENTEnabled:     "true",
				EnvSAPIENTHTTPURL:     "https://example.test/sapient",
				EnvSAPIENTRawMaxBytes: "0",
			},
			want: EnvSAPIENTRawMaxBytes,
		},
		{
			name: "zero sapient max response bytes",
			env: map[string]string{
				EnvSAPIENTEnabled:              "true",
				EnvSAPIENTHTTPURL:              "https://example.test/sapient",
				EnvSAPIENTHTTPMaxResponseBytes: "0",
			},
			want: EnvSAPIENTHTTPMaxResponseBytes,
		},
		{
			name: "zero sapient stale after",
			env: map[string]string{
				EnvSAPIENTEnabled:        "true",
				EnvSAPIENTHTTPURL:        "https://example.test/sapient",
				EnvSAPIENTHTTPStaleAfter: "0s",
			},
			want: EnvSAPIENTHTTPStaleAfter,
		},
		{
			name: "sapient graph enabled without sapient enabled",
			env: map[string]string{
				EnvSAPIENTGraphEnabled: "true",
			},
			want: EnvSAPIENTGraphEnabled,
		},
		{
			name: "zero sapient graph write timeout",
			env: map[string]string{
				EnvSAPIENTEnabled:      "true",
				EnvSAPIENTGraphEnabled: "true",
				EnvSAPIENTHTTPURL:      "https://example.test/sapient",
				EnvSAPIENTWriteTimeout: "0s",
			},
			want: EnvSAPIENTWriteTimeout,
		},
		{
			name: "sapient enabled without url",
			env:  map[string]string{EnvSAPIENTEnabled: "true"},
			want: EnvSAPIENTHTTPURL,
		},
		{
			name: "bad sapient encoding",
			env: map[string]string{
				EnvSAPIENTEnabled:      "true",
				EnvSAPIENTHTTPURL:      "https://example.test/sapient",
				EnvSAPIENTHTTPEncoding: "yaml",
			},
			want: EnvSAPIENTHTTPEncoding,
		},
		{
			name: "bad klv enabled",
			env:  map[string]string{EnvKLVEnabled: "sometimes"},
			want: EnvKLVEnabled,
		},
		{
			name: "bad klv write timeout",
			env:  map[string]string{EnvKLVWriteTimeout: "forever"},
			want: EnvKLVWriteTimeout,
		},
		{
			name: "bad klv demux max packet bytes",
			env:  map[string]string{EnvKLVDemuxMaxPacketBytes: "huge"},
			want: EnvKLVDemuxMaxPacketBytes,
		},
		{
			name: "bad klv demux max extract bytes",
			env:  map[string]string{EnvKLVDemuxMaxExtractBytes: "huge"},
			want: EnvKLVDemuxMaxExtractBytes,
		},
		{
			name: "bad klv demux max packets",
			env:  map[string]string{EnvKLVDemuxMaxPackets: "many"},
			want: EnvKLVDemuxMaxPackets,
		},
		{
			name: "bad klv demux max materialized bytes",
			env:  map[string]string{EnvKLVDemuxMaxMaterializedBytes: "huge"},
			want: EnvKLVDemuxMaxMaterializedBytes,
		},
		{
			name: "bad klv demux probe output max bytes",
			env:  map[string]string{EnvKLVDemuxProbeOutputMaxBytes: "huge"},
			want: EnvKLVDemuxProbeOutputMaxBytes,
		},
		{
			name: "bad klv decode max packet bytes",
			env:  map[string]string{EnvKLVDecodeMaxPacketBytes: "huge"},
			want: EnvKLVDecodeMaxPacketBytes,
		},
		{
			name: "zero klv max extract bytes",
			env: map[string]string{
				EnvKLVEnabled:              "true",
				EnvKLVDemuxMaxExtractBytes: "0",
			},
			want: EnvKLVDemuxMaxExtractBytes,
		},
		{
			name: "klv max extract bytes below max packet bytes",
			env: map[string]string{
				EnvKLVEnabled:              "true",
				EnvKLVDemuxMaxPacketBytes:  "128",
				EnvKLVDemuxMaxExtractBytes: "64",
			},
			want: EnvKLVDemuxMaxExtractBytes,
		},
		{
			name: "zero klv max packets",
			env: map[string]string{
				EnvKLVEnabled:         "true",
				EnvKLVDemuxMaxPackets: "0",
			},
			want: EnvKLVDemuxMaxPackets,
		},
		{
			name: "zero klv decode max packet bytes",
			env: map[string]string{
				EnvKLVEnabled:              "true",
				EnvKLVDecodeMaxPacketBytes: "0",
			},
			want: EnvKLVDecodeMaxPacketBytes,
		},
		{
			name: "bad weather enabled",
			env:  map[string]string{EnvWeatherEnabled: "sometimes"},
			want: EnvWeatherEnabled,
		},
		{
			name: "bad weather write timeout",
			env:  map[string]string{EnvWeatherWriteTimeout: "forever"},
			want: EnvWeatherWriteTimeout,
		},
		{
			name: "bad weather freshness",
			env:  map[string]string{EnvWeatherFreshness: "forever"},
			want: EnvWeatherFreshness,
		},
		{
			name: "bad weather max observations",
			env:  map[string]string{EnvWeatherMaxObservations: "many"},
			want: EnvWeatherMaxObservations,
		},
		{
			name: "zero weather freshness",
			env: map[string]string{
				EnvWeatherEnabled:   "true",
				EnvWeatherFreshness: "0s",
			},
			want: EnvWeatherFreshness,
		},
		{
			name: "zero weather max observations",
			env: map[string]string{
				EnvWeatherEnabled:         "true",
				EnvWeatherMaxObservations: "0",
			},
			want: EnvWeatherMaxObservations,
		},
		{
			name: "bad fusion enabled",
			env:  map[string]string{EnvFusionEnabled: "sometimes"},
			want: EnvFusionEnabled,
		},
		{
			name: "bad fusion write timeout",
			env:  map[string]string{EnvFusionWriteTimeout: "eventually"},
			want: EnvFusionWriteTimeout,
		},
		{
			name: "zero fusion write timeout",
			env: map[string]string{
				EnvFusionEnabled:      "true",
				EnvFusionWriteTimeout: "0s",
			},
			want: EnvFusionWriteTimeout,
		},
		{
			name: "bad fusion association max distance",
			env:  map[string]string{EnvFusionAssociationMaxDistance: "far"},
			want: EnvFusionAssociationMaxDistance,
		},
		{
			name: "zero fusion association max distance",
			env: map[string]string{
				EnvFusionEnabled:                "true",
				EnvFusionAssociationMaxDistance: "0",
			},
			want: EnvFusionAssociationMaxDistance,
		},
		{
			name: "bad fusion association max time delta",
			env:  map[string]string{EnvFusionAssociationMaxTimeDelta: "soon"},
			want: EnvFusionAssociationMaxTimeDelta,
		},
		{
			name: "zero fusion association max time delta",
			env: map[string]string{
				EnvFusionEnabled:                 "true",
				EnvFusionAssociationMaxTimeDelta: "0s",
			},
			want: EnvFusionAssociationMaxTimeDelta,
		},
		{
			name: "bad fusion association max observation age",
			env:  map[string]string{EnvFusionAssociationMaxObservationAge: "old"},
			want: EnvFusionAssociationMaxObservationAge,
		},
		{
			name: "zero fusion association max observation age",
			env: map[string]string{
				EnvFusionEnabled:                      "true",
				EnvFusionAssociationMaxObservationAge: "0s",
			},
			want: EnvFusionAssociationMaxObservationAge,
		},
		{
			name: "empty fusion association source priority",
			env: map[string]string{
				EnvFusionEnabled:                   "true",
				EnvFusionAssociationSourcePriority: ",",
			},
			want: EnvFusionAssociationSourcePriority,
		},
		{
			name: "bad fusion association min confidence",
			env:  map[string]string{EnvFusionAssociationMinConfidence: "certain"},
			want: EnvFusionAssociationMinConfidence,
		},
		{
			name: "out of range fusion association min confidence",
			env: map[string]string{
				EnvFusionEnabled:                  "true",
				EnvFusionAssociationMinConfidence: "1.2",
			},
			want: EnvFusionAssociationMinConfidence,
		},
		{
			name: "bad fusion association ambiguity margin",
			env:  map[string]string{EnvFusionAssociationAmbiguityMargin: "wide"},
			want: EnvFusionAssociationAmbiguityMargin,
		},
		{
			name: "out of range fusion association ambiguity margin",
			env: map[string]string{
				EnvFusionEnabled:                    "true",
				EnvFusionAssociationAmbiguityMargin: "0",
			},
			want: EnvFusionAssociationAmbiguityMargin,
		},
		{
			name: "bad fusion candidates enabled",
			env:  map[string]string{EnvFusionCandidatesEnabled: "sometimes"},
			want: EnvFusionCandidatesEnabled,
		},
		{
			name: "bad fusion candidate poll interval",
			env:  map[string]string{EnvFusionCandidatePollInterval: "soon"},
			want: EnvFusionCandidatePollInterval,
		},
		{
			name: "bad fusion candidate query timeout",
			env:  map[string]string{EnvFusionCandidateQueryTimeout: "eventually"},
			want: EnvFusionCandidateQueryTimeout,
		},
		{
			name: "bad fusion candidate limit per source",
			env:  map[string]string{EnvFusionCandidateLimitPerSource: "many"},
			want: EnvFusionCandidateLimitPerSource,
		},
		{
			name: "bad fusion candidate max comparisons",
			env:  map[string]string{EnvFusionCandidateMaxComparisons: "many"},
			want: EnvFusionCandidateMaxComparisons,
		},
		{
			name: "bad fusion candidate max batches",
			env:  map[string]string{EnvFusionCandidateMaxBatches: "many"},
			want: EnvFusionCandidateMaxBatches,
		},
		{
			name: "empty fusion candidate sources",
			env: map[string]string{
				EnvFusionCandidatesEnabled: "true",
				EnvFusionCandidateSources:  ",",
			},
			want: EnvFusionCandidateSources,
		},
		{
			name: "zero fusion candidate poll interval",
			env: map[string]string{
				EnvFusionCandidatesEnabled:     "true",
				EnvFusionCandidatePollInterval: "0s",
			},
			want: EnvFusionCandidatePollInterval,
		},
		{
			name: "zero fusion candidate query timeout",
			env: map[string]string{
				EnvFusionCandidatesEnabled:     "true",
				EnvFusionCandidateQueryTimeout: "0s",
			},
			want: EnvFusionCandidateQueryTimeout,
		},
		{
			name: "zero fusion candidate limit per source",
			env: map[string]string{
				EnvFusionCandidatesEnabled:       "true",
				EnvFusionCandidateLimitPerSource: "0",
			},
			want: EnvFusionCandidateLimitPerSource,
		},
		{
			name: "zero fusion candidate max comparisons",
			env: map[string]string{
				EnvFusionCandidatesEnabled:       "true",
				EnvFusionCandidateMaxComparisons: "0",
			},
			want: EnvFusionCandidateMaxComparisons,
		},
		{
			name: "zero fusion candidate max batches",
			env: map[string]string{
				EnvFusionCandidatesEnabled:   "true",
				EnvFusionCandidateMaxBatches: "0",
			},
			want: EnvFusionCandidateMaxBatches,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ConfigFromEnv(func(name string) string {
				return tt.env[name]
			})
			if err == nil {
				t.Fatal("expected config error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want env name %s", err, tt.want)
			}
		})
	}
}

type fakeSemStreamsClient struct {
	mu            sync.Mutex
	connected     bool
	closed        bool
	subscriptions map[string][]func(context.Context, *nats.Msg)
	published     []publishedRuntimeMessage
	requests      []runtimeGraphRequest
}

func (c *fakeSemStreamsClient) Connect(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = true
	return nil
}

func (c *fakeSemStreamsClient) Close(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *fakeSemStreamsClient) RequestWithRetry(
	_ context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
	retry natsclient.RetryConfig,
) ([]byte, error) {
	c.mu.Lock()
	c.requests = append(c.requests, runtimeGraphRequest{
		subject: subject,
		data:    append([]byte(nil), data...),
		timeout: timeout,
		retry:   retry,
	})
	c.mu.Unlock()

	switch subject {
	case mavprojector.SubjectEntityCreateWithTriples:
		return mustRuntimeJSON(graph.CreateEntityWithTriplesResponse{
			MutationResponse: graph.MutationResponse{},
		}), nil
	case mavprojector.SubjectEntityUpdateWithTriples:
		return mustRuntimeJSON(graph.UpdateEntityWithTriplesResponse{
			MutationResponse: graph.MutationResponse{},
		}), nil
	default:
		return nil, errors.New("unexpected request subject: " + subject)
	}
}

func (c *fakeSemStreamsClient) RequestWithRetryClassified(
	ctx context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
	retry natsclient.RetryConfig,
) ([]byte, error) {
	return c.RequestWithRetry(ctx, subject, data, timeout, retry)
}

func (c *fakeSemStreamsClient) Request(
	ctx context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
) ([]byte, error) {
	return c.RequestWithRetry(ctx, subject, data, timeout, natsclient.DefaultRetryConfig())
}

func (c *fakeSemStreamsClient) RequestClassified(context.Context, string, []byte, time.Duration) ([]byte, error) {
	return nil, errors.New("not used")
}

func (c *fakeSemStreamsClient) Publish(ctx context.Context, subject string, data []byte) error {
	c.mu.Lock()
	c.published = append(c.published, publishedRuntimeMessage{
		subject: subject,
		data:    append([]byte(nil), data...),
	})
	handlers := append([]func(context.Context, *nats.Msg){}, c.subscriptions[subject]...)
	c.mu.Unlock()

	for _, handler := range handlers {
		handler(ctx, &nats.Msg{Subject: subject, Data: append([]byte(nil), data...)})
	}
	return nil
}

func (c *fakeSemStreamsClient) Subscribe(
	_ context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (*natsclient.Subscription, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.subscriptions == nil {
		c.subscriptions = make(map[string][]func(context.Context, *nats.Msg))
	}
	c.subscriptions[subject] = append(c.subscriptions[subject], handler)
	return nil, nil
}

func (c *fakeSemStreamsClient) hasSubscription(subject string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.subscriptions[subject]) > 0
}

func (c *fakeSemStreamsClient) subscriptionSubjects() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	subjects := make([]string, 0, len(c.subscriptions))
	for subject := range c.subscriptions {
		subjects = append(subjects, subject)
	}
	return subjects
}

func (c *fakeSemStreamsClient) requestCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.requests)
}

func (c *fakeSemStreamsClient) publishedCount(subject string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	count := 0
	for _, msg := range c.published {
		if msg.subject == subject {
			count++
		}
	}
	return count
}

func pollUntil(t *testing.T, timeout time.Duration, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition was not satisfied before %s", timeout)
}

type publishedRuntimeMessage struct {
	subject string
	data    []byte
}

type runtimeGraphRequest struct {
	subject string
	data    []byte
	timeout time.Duration
	retry   natsclient.RetryConfig
}

type recordingAppPlanWriter struct {
	plans []mavprojector.Plan
}

func (w *recordingAppPlanWriter) Apply(_ context.Context, plan mavprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return nil
}

type recordingCoTAppPlanWriter struct {
	plans []cotprojector.Plan
}

func (w *recordingCoTAppPlanWriter) Apply(_ context.Context, plan cotprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return nil
}

type recordingCAPAppPlanWriter struct {
	plans []capprojector.Plan
}

func (w *recordingCAPAppPlanWriter) Apply(_ context.Context, plan capprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return nil
}

type recordingADSBAppPlanWriter struct {
	plans []adsbprojector.Plan
}

func (w *recordingADSBAppPlanWriter) Apply(_ context.Context, plan adsbprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return nil
}

type recordingSAPIENTAppPlanWriter struct {
	plans []sapientprojector.Plan
}

func (w *recordingSAPIENTAppPlanWriter) Apply(_ context.Context, plan sapientprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return nil
}

type recordingKLVAppPlanWriter struct {
	plans []klvprojector.Plan
}

func (w *recordingKLVAppPlanWriter) Apply(_ context.Context, plan klvprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return nil
}

type recordingWeatherAppPlanWriter struct {
	plans []weatherprojector.Plan
}

func (w *recordingWeatherAppPlanWriter) Apply(_ context.Context, plan weatherprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return nil
}

type recordingFusionAppPlanWriter struct {
	plans []fusionprojector.Plan
}

func (w *recordingFusionAppPlanWriter) Apply(_ context.Context, plan fusionprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return nil
}

func hasOwnedContract(owned []cop.OwnedContract, owner string, contractName string) bool {
	for _, item := range owned {
		if item.Owner == owner && item.Contract.Name == contractName {
			return true
		}
	}
	return false
}

func hasComponentMetricSource(sources []componentmetrics.Source, name, feed, role string) bool {
	for _, source := range sources {
		if source.Component == nil {
			continue
		}
		meta := source.Component.Meta()
		if meta.Name == name && source.Feed == feed && source.Role == role {
			return true
		}
	}
	return false
}

func componentMetricSourceNames(sources []componentmetrics.Source) []string {
	names := make([]string, 0, len(sources))
	for _, source := range sources {
		if source.Component == nil {
			continue
		}
		names = append(names, source.Feed+"/"+source.Role+"/"+source.Component.Meta().Name)
	}
	return names
}

func mustRuntimeHeartbeatFrame(t *testing.T) []byte {
	t.Helper()
	frame, err := mavcodec.NewGenerator(42, 7).GenerateHeartbeat(mavcodec.HeartbeatMessage{
		BaseMode:       mavcodec.ModeFlagSafetyArmed,
		SystemStatus:   mavcodec.StateActive,
		MavlinkVersion: mavcodec.Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	return frame
}

func mustRuntimeCoTEvent(t *testing.T) []byte {
	t.Helper()
	now := time.Now().UTC()
	raw, err := cotcodec.Marshal(cotcodec.Event{
		UID:      "ANDROID-ALPHA",
		Type:     cotcodec.TypeOperatorPosition,
		How:      cotcodec.DefaultHow,
		Time:     now,
		Start:    now,
		Stale:    now.Add(2 * time.Minute),
		Callsign: "Alpha Team",
		Point: &cotcodec.Point{
			Lat: 30.2672,
			Lon: -97.7431,
			HAE: 188,
			CE:  5,
			LE:  8,
		},
	})
	if err != nil {
		t.Fatalf("marshal CoT event: %v", err)
	}
	return raw
}

func mustRuntimeJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}

const runtimeSAPIENTTaskAckJSON = `{
  "timestamp": "2023-07-07T12:44:17.027638700Z",
  "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c8",
  "taskAck": {
    "taskId": "01H4R63D7NVN8444Z5M77WEBY8",
    "taskStatus": "TASK_STATUS_ACCEPTED",
    "reason": ["accepted for runtime preflight"]
  }
}`
