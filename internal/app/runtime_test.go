package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	mavadapter "github.com/c360studio/semops/internal/adapters/mavlink"
	"github.com/c360studio/semops/internal/copownership"
	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	"github.com/c360studio/semops/internal/stack"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestStartRegistersOwnershipBeforeComposingMAVLink(t *testing.T) {
	ctx := context.Background()
	client := &fakeSemStreamsClient{}
	cfg := DefaultConfig()
	cfg.NATSURL = "nats://semstreams:4222"
	cfg.MAVLink.Platform = "edge-alpha"
	cfg.MAVLink.Source = "udp:14550"
	cfg.MAVLink.WriteTimeout = 750 * time.Millisecond

	var stoppedOwners bool
	var gotAdapterCfg stack.MAVLinkAdapterConfig
	var gotAdapterDeps stack.MAVLinkAdapterDeps

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
		newMAVLinkAdapter: func(
			adapterCfg stack.MAVLinkAdapterConfig,
			adapterDeps stack.MAVLinkAdapterDeps,
		) (*mavadapter.Adapter, error) {
			gotAdapterCfg = adapterCfg
			gotAdapterDeps = adapterDeps
			return testMAVLinkAdapter(t), nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if app.OwnershipBinding().Incarnation != "lease-123" {
		t.Fatalf("ownership incarnation = %q", app.OwnershipBinding().Incarnation)
	}
	if app.MAVLinkAdapter() == nil {
		t.Fatal("expected hosted MAVLink adapter")
	}
	if got, want := gotAdapterCfg.OwnerTokens[cop.OwnerMAVLink].Wire(), "semops.feed.mavlink#lease-123"; got != want {
		t.Fatalf("adapter MAVLink owner token = %q, want %q", got, want)
	}
	if gotAdapterCfg.Platform != "edge-alpha" || gotAdapterCfg.Source != "udp:14550" {
		t.Fatalf("adapter config = %+v", gotAdapterCfg)
	}
	if gotAdapterCfg.WriteTimeout != 750*time.Millisecond {
		t.Fatalf("adapter write timeout = %s", gotAdapterCfg.WriteTimeout)
	}
	if gotAdapterDeps.NATS != client {
		t.Fatal("MAVLink adapter must reuse the connected SemStreams client")
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
		newMAVLinkAdapter: func(
			stack.MAVLinkAdapterConfig,
			stack.MAVLinkAdapterDeps,
		) (*mavadapter.Adapter, error) {
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

func TestStartCanDisableHostedMAVLinkAdapter(t *testing.T) {
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
		newMAVLinkAdapter: func(
			stack.MAVLinkAdapterConfig,
			stack.MAVLinkAdapterDeps,
		) (*mavadapter.Adapter, error) {
			composed = true
			return testMAVLinkAdapter(t), nil
		},
	})
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if composed {
		t.Fatal("MAVLink adapter should not be composed when disabled")
	}
	if app.MAVLinkAdapter() != nil {
		t.Fatal("MAVLink adapter should be nil when disabled")
	}
}

func TestConfigFromEnv(t *testing.T) {
	env := map[string]string{
		EnvNATSURL:                    "nats://semstreams:4222",
		EnvNATSName:                   "semops-test",
		EnvNATSConnectTimeout:         "3s",
		EnvOwnershipHeartbeatInterval: "4s",
		EnvMAVLinkEnabled:             "false",
		EnvMAVLinkSource:              "udp:14550",
		EnvOrg:                        "lab",
		EnvPlatform:                   "edge-7",
		EnvTraceID:                    "trace-7",
		EnvMAVLinkWriteTimeout:        "900ms",
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
	if cfg.OwnershipHeartbeatInterval != 4*time.Second {
		t.Fatalf("heartbeat interval = %s", cfg.OwnershipHeartbeatInterval)
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
}

func TestConfigFromEnvReportsBadValues(t *testing.T) {
	_, err := ConfigFromEnv(func(name string) string {
		if name == EnvMAVLinkWriteTimeout {
			return "forever"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), EnvMAVLinkWriteTimeout) {
		t.Fatalf("error = %v, want env name", err)
	}
}

type fakeSemStreamsClient struct {
	connected bool
	closed    bool
}

func (c *fakeSemStreamsClient) Connect(context.Context) error {
	c.connected = true
	return nil
}

func (c *fakeSemStreamsClient) Close(context.Context) error {
	c.closed = true
	return nil
}

func (c *fakeSemStreamsClient) RequestWithRetry(
	context.Context,
	string,
	[]byte,
	time.Duration,
	natsclient.RetryConfig,
) ([]byte, error) {
	return nil, errors.New("not used")
}

type noopPlanWriter struct{}

func (noopPlanWriter) Apply(context.Context, mavprojector.Plan) error {
	return nil
}

func testMAVLinkAdapter(t *testing.T) *mavadapter.Adapter {
	t.Helper()
	adapter, err := mavadapter.NewAdapter(mavadapter.Config{Writer: noopPlanWriter{}})
	if err != nil {
		t.Fatalf("new test adapter: %v", err)
	}
	return adapter
}
