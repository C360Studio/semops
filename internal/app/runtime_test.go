package app

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	cotadapter "github.com/c360studio/semops/internal/adapters/cot"
	mavcomponent "github.com/c360studio/semops/internal/components/mavlink"
	"github.com/c360studio/semops/internal/copownership"
	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	"github.com/c360studio/semops/internal/stack"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semops/pkg/cop"
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
	if !client.hasSubscription(mavcomponent.DefaultDecodedSubject) || !client.hasSubscription(mavcomponent.DefaultRawSubject) {
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

func TestStartComposesCoTAdapterAfterOwnershipRegistration(t *testing.T) {
	ctx := context.Background()
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.CoT.Enabled = true
	cfg.CoT.Platform = "edge-alpha"
	cfg.CoT.Source = "udp:cot"
	cfg.CoT.WriteTimeout = 750 * time.Millisecond

	var stoppedOwners bool
	var gotAdapterCfg stack.CoTAdapterConfig
	var gotAdapterDeps stack.CoTAdapterDeps

	app, err := start(ctx, cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			_ context.Context,
			gotClient semstreamsClient,
			heartbeat time.Duration,
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
		newCoTAdapter: func(
			adapterCfg stack.CoTAdapterConfig,
			adapterDeps stack.CoTAdapterDeps,
		) (*cotadapter.Adapter, error) {
			gotAdapterCfg = adapterCfg
			gotAdapterDeps = adapterDeps
			return testCoTAdapter(t), nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.CoTAdapter() == nil {
		t.Fatal("expected hosted CoT adapter")
	}
	if got, want := gotAdapterCfg.OwnerTokens[cop.OwnerTAK].Wire(), "semops.feed.tak#lease-123"; got != want {
		t.Fatalf("adapter TAK owner token = %q, want %q", got, want)
	}
	if gotAdapterCfg.Platform != "edge-alpha" || gotAdapterCfg.Source != "udp:cot" {
		t.Fatalf("adapter config = %+v", gotAdapterCfg)
	}
	if gotAdapterCfg.WriteTimeout != 750*time.Millisecond {
		t.Fatalf("adapter write timeout = %s", gotAdapterCfg.WriteTimeout)
	}
	if gotAdapterDeps.NATS != client {
		t.Fatal("CoT adapter must reuse the connected SemStreams client")
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
	adapterErr := errors.New("bad cot adapter")
	var stoppedOwners bool

	_, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {
				stoppedOwners = true
			}, nil
		},
		newCoTAdapter: func(
			stack.CoTAdapterConfig,
			stack.CoTAdapterDeps,
		) (*cotadapter.Adapter, error) {
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

func TestStartCanDisableHostedCoTAdapter(t *testing.T) {
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
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
		newCoTAdapter: func(
			stack.CoTAdapterConfig,
			stack.CoTAdapterDeps,
		) (*cotadapter.Adapter, error) {
			composed = true
			return testCoTAdapter(t), nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if composed {
		t.Fatal("CoT adapter should not be composed when disabled")
	}
	if app.CoTAdapter() != nil {
		t.Fatal("CoT adapter should be nil when disabled")
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
	if !client.hasSubscription(mavcomponent.DefaultRawSubject) || !client.hasSubscription(mavcomponent.DefaultDecodedSubject) {
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

func TestStartHostsCoTTransportsWhenConfigured(t *testing.T) {
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.MAVLink.Enabled = false
	cfg.CoT.Enabled = true
	cfg.CoT.UDP.ListenAddr = "127.0.0.1:18080"
	cfg.CoT.UDP.MaxDatagramBytes = 2048
	cfg.CoT.TCP.ListenAddr = "127.0.0.1:18081"
	cfg.CoT.TCP.MaxEventBytes = 4096
	udpTransport := newFakeMAVLinkTransport()
	tcpTransport := newFakeMAVLinkTransport()
	var gotUDPCfg cotadapter.UDPListenerConfig
	var gotTCPCfg cotadapter.TCPListenerConfig
	var gotUDPAdapter *cotadapter.Adapter
	var gotTCPAdapter *cotadapter.Adapter

	app, err := start(context.Background(), cfg, dependencies{
		newNATSClient: func(Config) (semstreamsClient, error) {
			return client, nil
		},
		registerOwners: func(
			context.Context,
			semstreamsClient,
			time.Duration,
		) (copownership.BindingResult, func(), error) {
			return copownership.BindingResult{Incarnation: "lease-123"}, func() {}, nil
		},
		newCoTAdapter: func(
			stack.CoTAdapterConfig,
			stack.CoTAdapterDeps,
		) (*cotadapter.Adapter, error) {
			return testCoTAdapter(t), nil
		},
		newCoTUDPListener: func(
			transportCfg cotadapter.UDPListenerConfig,
			adapter *cotadapter.Adapter,
		) (cotTransport, error) {
			gotUDPCfg = transportCfg
			gotUDPAdapter = adapter
			return udpTransport, nil
		},
		newCoTTCPListener: func(
			transportCfg cotadapter.TCPListenerConfig,
			adapter *cotadapter.Adapter,
		) (cotTransport, error) {
			gotTCPCfg = transportCfg
			gotTCPAdapter = adapter
			return tcpTransport, nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	udpTransport.waitStarted(t)
	tcpTransport.waitStarted(t)
	if gotUDPCfg.ListenAddr != "127.0.0.1:18080" ||
		gotUDPCfg.MaxDatagramBytes != 2048 {
		t.Fatalf("CoT UDP config = %+v", gotUDPCfg)
	}
	if gotTCPCfg.ListenAddr != "127.0.0.1:18081" ||
		gotTCPCfg.MaxEventBytes != 4096 {
		t.Fatalf("CoT TCP config = %+v", gotTCPCfg)
	}
	if gotUDPAdapter != app.CoTAdapter() || gotTCPAdapter != app.CoTAdapter() {
		t.Fatal("CoT transports must receive the hosted CoT adapter")
	}

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close app: %v", err)
	}
	udpTransport.waitClosed(t)
	tcpTransport.waitClosed(t)
	if !client.closed {
		t.Fatal("close must close SemStreams client")
	}
}

func TestConfigFromEnv(t *testing.T) {
	env := map[string]string{
		EnvNATSURL:                    "nats://semstreams:4222",
		EnvNATSName:                   "semops-test",
		EnvNATSConnectTimeout:         "3s",
		EnvAPIAddr:                    ":18088",
		EnvOwnershipHeartbeatInterval: "4s",
		EnvCOPGraphQueryTimeout:       "1500ms",
		EnvCOPGraphDiscoveryEnabled:   "false",
		EnvCOPGraphDiscoveryLimit:     "75",
		EnvCOPMAVLinkSystemIDs:        "42, 43",
		EnvCOPCoTUIDs:                 "ANDROID-ALPHA, MARKER-NORTH-GATE",
		EnvCOPCAPAlertIDs:             "nws-demo-flood-warning, nws-demo-flood-update",
		EnvMAVLinkEnabled:             "false",
		EnvMAVLinkSource:              "udp:14550",
		EnvOrg:                        "lab",
		EnvPlatform:                   "edge-7",
		EnvTraceID:                    "trace-7",
		EnvMAVLinkWriteTimeout:        "900ms",
		EnvMAVLinkUDPListenAddr:       "127.0.0.1:14550",
		EnvMAVLinkUDPMaxDatagramBytes: "2048",
		EnvCoTEnabled:                 "true",
		EnvCoTSource:                  "udp:cot",
		EnvCoTWriteTimeout:            "950ms",
		EnvCoTUDPListenAddr:           "127.0.0.1:18080",
		EnvCoTUDPMaxDatagramBytes:     "4096",
		EnvCoTTCPListenAddr:           "127.0.0.1:18081",
		EnvCoTTCPMaxEventBytes:        "8192",
	}

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
			MutationResponse: graph.MutationResponse{Success: true},
		}), nil
	case mavprojector.SubjectEntityUpdateWithTriples:
		return mustRuntimeJSON(graph.UpdateEntityWithTriplesResponse{
			MutationResponse: graph.MutationResponse{Success: true},
		}), nil
	default:
		return nil, errors.New("unexpected request subject: " + subject)
	}
}

func (c *fakeSemStreamsClient) Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error) {
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

type noopCoTPlanWriter struct{}

func (noopCoTPlanWriter) Apply(context.Context, cotprojector.Plan) error {
	return nil
}

func testCoTAdapter(t *testing.T) *cotadapter.Adapter {
	t.Helper()
	adapter, err := cotadapter.NewAdapter(cotadapter.Config{
		Projector: cotprojector.NewProjector(cotprojector.Config{}),
		Writer:    noopCoTPlanWriter{},
	})
	if err != nil {
		t.Fatalf("new test CoT adapter: %v", err)
	}
	return adapter
}

type fakeMAVLinkTransport struct {
	started   chan struct{}
	closed    chan struct{}
	closeOnce sync.Once
}

func newFakeMAVLinkTransport() *fakeMAVLinkTransport {
	return &fakeMAVLinkTransport{
		started: make(chan struct{}),
		closed:  make(chan struct{}),
	}
}

func (t *fakeMAVLinkTransport) Run(ctx context.Context) error {
	close(t.started)
	<-ctx.Done()
	return nil
}

func (t *fakeMAVLinkTransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.closed)
	})
	return nil
}

func (t *fakeMAVLinkTransport) waitStarted(tb testing.TB) {
	tb.Helper()
	select {
	case <-t.started:
	case <-time.After(time.Second):
		tb.Fatal("timed out waiting for transport start")
	}
}

func (t *fakeMAVLinkTransport) waitClosed(tb testing.TB) {
	tb.Helper()
	select {
	case <-t.closed:
	case <-time.After(time.Second):
		tb.Fatal("timed out waiting for transport close")
	}
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
