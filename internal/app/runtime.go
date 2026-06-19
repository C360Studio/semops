package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	cotadapter "github.com/c360studio/semops/internal/adapters/cot"
	mavadapter "github.com/c360studio/semops/internal/adapters/mavlink"
	"github.com/c360studio/semops/internal/copownership"
	"github.com/c360studio/semops/internal/graphrequest"
	"github.com/c360studio/semops/internal/stack"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

type App struct {
	client           semstreamsClient
	ownershipStop    func()
	ownershipResult  copownership.BindingResult
	mavlinkAdapter   *mavadapter.Adapter
	mavlinkTransport mavlinkTransport
	cotAdapter       *cotadapter.Adapter
	cotTransports    []runningCoTTransport
	transportCancel  context.CancelFunc
	transportDone    chan error
}

type semstreamsClient interface {
	graphrequest.RetryRequester
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	RequestClassified(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	Connect(context.Context) error
	Close(context.Context) error
}

type GraphRequester interface {
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	RequestClassified(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
}

type dependencies struct {
	newNATSClient         func(Config) (semstreamsClient, error)
	registerOwners        func(context.Context, semstreamsClient, time.Duration) (copownership.BindingResult, func(), error)
	newMAVLinkAdapter     func(stack.MAVLinkAdapterConfig, stack.MAVLinkAdapterDeps) (*mavadapter.Adapter, error)
	newMAVLinkUDPListener func(mavadapter.UDPListenerConfig, *mavadapter.Adapter) (mavlinkTransport, error)
	newCoTAdapter         func(stack.CoTAdapterConfig, stack.CoTAdapterDeps) (*cotadapter.Adapter, error)
	newCoTUDPListener     func(cotadapter.UDPListenerConfig, *cotadapter.Adapter) (cotTransport, error)
	newCoTTCPListener     func(cotadapter.TCPListenerConfig, *cotadapter.Adapter) (cotTransport, error)
}

type mavlinkTransport interface {
	Run(context.Context) error
	Close() error
}

type cotTransport interface {
	Run(context.Context) error
	Close() error
}

type runningCoTTransport struct {
	name      string
	transport cotTransport
	cancel    context.CancelFunc
	done      chan error
}

func Start(ctx context.Context, cfg Config) (*App, error) {
	return start(ctx, cfg, defaultDependencies())
}

func (a *App) Close(ctx context.Context) error {
	if a == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var errs []error
	if a.transportCancel != nil {
		a.transportCancel()
	}
	for _, running := range a.cotTransports {
		if running.cancel != nil {
			running.cancel()
		}
	}
	if a.mavlinkTransport != nil {
		if err := a.mavlinkTransport.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close MAVLink transport: %w", err))
		}
	}
	for _, running := range a.cotTransports {
		if running.transport != nil {
			if err := running.transport.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close %s transport: %w", running.name, err))
			}
		}
	}
	if a.transportDone != nil {
		select {
		case err := <-a.transportDone:
			if err != nil {
				errs = append(errs, fmt.Errorf("run MAVLink transport: %w", err))
			}
		case <-ctx.Done():
			errs = append(errs, fmt.Errorf("wait for MAVLink transport shutdown: %w", ctx.Err()))
		}
	}
	for _, running := range a.cotTransports {
		if running.done == nil {
			continue
		}
		select {
		case err := <-running.done:
			if err != nil {
				errs = append(errs, fmt.Errorf("run %s transport: %w", running.name, err))
			}
		case <-ctx.Done():
			errs = append(errs, fmt.Errorf("wait for %s transport shutdown: %w", running.name, ctx.Err()))
		}
	}
	if a.ownershipStop != nil {
		a.ownershipStop()
	}
	if a.client != nil {
		if err := a.client.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (a *App) OwnershipBinding() copownership.BindingResult {
	if a == nil {
		return copownership.BindingResult{}
	}
	return a.ownershipResult
}

func (a *App) MAVLinkAdapter() *mavadapter.Adapter {
	if a == nil {
		return nil
	}
	return a.mavlinkAdapter
}

func (a *App) CoTAdapter() *cotadapter.Adapter {
	if a == nil {
		return nil
	}
	return a.cotAdapter
}

func (a *App) GraphRequester() GraphRequester {
	if a == nil {
		return nil
	}
	return a.client
}

func start(ctx context.Context, cfg Config, deps dependencies) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	deps = fillDependencies(deps)

	client, err := deps.newNATSClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create SemStreams NATS client: %w", err)
	}
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connect SemStreams NATS: %w", err)
	}

	app := &App{client: client}
	cleanup := true
	defer func() {
		if cleanup {
			_ = app.Close(context.Background())
		}
	}()

	bindings, stopOwners, err := deps.registerOwners(ctx, client, cfg.OwnershipHeartbeatInterval)
	if err != nil {
		return nil, fmt.Errorf("register SemOps COP ownership: %w", err)
	}
	app.ownershipResult = bindings
	app.ownershipStop = stopOwners

	if cfg.MAVLink.Enabled {
		adapter, err := deps.newMAVLinkAdapter(stack.MAVLinkAdapterConfig{
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
			return nil, fmt.Errorf("compose MAVLink adapter: %w", err)
		}
		app.mavlinkAdapter = adapter
		if cfg.MAVLink.UDP.ListenAddr != "" {
			if err := app.startMAVLinkUDPTransport(cfg.MAVLink.UDP, deps, adapter); err != nil {
				return nil, err
			}
		}
	}

	if cfg.CoT.Enabled {
		adapter, err := deps.newCoTAdapter(stack.CoTAdapterConfig{
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
			return nil, fmt.Errorf("compose CoT adapter: %w", err)
		}
		app.cotAdapter = adapter
		if cfg.CoT.UDP.ListenAddr != "" {
			transport, err := deps.newCoTUDPListener(cotadapter.UDPListenerConfig{
				ListenAddr:       cfg.CoT.UDP.ListenAddr,
				MaxDatagramBytes: cfg.CoT.UDP.MaxDatagramBytes,
			}, adapter)
			if err != nil {
				return nil, fmt.Errorf("start CoT UDP listener: %w", err)
			}
			app.startCoTTransport("CoT UDP", transport)
		}
		if cfg.CoT.TCP.ListenAddr != "" {
			transport, err := deps.newCoTTCPListener(cotadapter.TCPListenerConfig{
				ListenAddr:    cfg.CoT.TCP.ListenAddr,
				MaxEventBytes: cfg.CoT.TCP.MaxEventBytes,
			}, adapter)
			if err != nil {
				return nil, fmt.Errorf("start CoT TCP listener: %w", err)
			}
			app.startCoTTransport("CoT TCP", transport)
		}
	}

	cleanup = false
	return app, nil
}

func defaultDependencies() dependencies {
	return dependencies{
		newNATSClient:         newNATSClient,
		registerOwners:        registerOwners,
		newMAVLinkAdapter:     stack.NewMAVLinkAdapter,
		newMAVLinkUDPListener: newMAVLinkUDPListener,
		newCoTAdapter:         stack.NewCoTAdapter,
		newCoTUDPListener:     newCoTUDPListener,
		newCoTTCPListener:     newCoTTCPListener,
	}
}

func fillDependencies(deps dependencies) dependencies {
	defaults := defaultDependencies()
	if deps.newNATSClient == nil {
		deps.newNATSClient = defaults.newNATSClient
	}
	if deps.registerOwners == nil {
		deps.registerOwners = defaults.registerOwners
	}
	if deps.newMAVLinkAdapter == nil {
		deps.newMAVLinkAdapter = defaults.newMAVLinkAdapter
	}
	if deps.newMAVLinkUDPListener == nil {
		deps.newMAVLinkUDPListener = defaults.newMAVLinkUDPListener
	}
	if deps.newCoTAdapter == nil {
		deps.newCoTAdapter = defaults.newCoTAdapter
	}
	if deps.newCoTUDPListener == nil {
		deps.newCoTUDPListener = defaults.newCoTUDPListener
	}
	if deps.newCoTTCPListener == nil {
		deps.newCoTTCPListener = defaults.newCoTTCPListener
	}
	return deps
}

func (a *App) startMAVLinkUDPTransport(
	cfg MAVLinkUDPConfig,
	deps dependencies,
	adapter *mavadapter.Adapter,
) error {
	transport, err := deps.newMAVLinkUDPListener(mavadapter.UDPListenerConfig{
		ListenAddr:       cfg.ListenAddr,
		MaxDatagramBytes: cfg.MaxDatagramBytes,
	}, adapter)
	if err != nil {
		return fmt.Errorf("start MAVLink UDP listener: %w", err)
	}

	transportCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	a.mavlinkTransport = transport
	a.transportCancel = cancel
	a.transportDone = done
	go func() {
		done <- transport.Run(transportCtx)
	}()
	return nil
}

func (a *App) startCoTTransport(name string, transport cotTransport) {
	transportCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	a.cotTransports = append(a.cotTransports, runningCoTTransport{
		name:      name,
		transport: transport,
		cancel:    cancel,
		done:      done,
	})
	go func() {
		done <- transport.Run(transportCtx)
	}()
}

func newMAVLinkUDPListener(
	cfg mavadapter.UDPListenerConfig,
	adapter *mavadapter.Adapter,
) (mavlinkTransport, error) {
	return mavadapter.ListenUDP(cfg, adapter)
}

func newCoTUDPListener(
	cfg cotadapter.UDPListenerConfig,
	adapter *cotadapter.Adapter,
) (cotTransport, error) {
	return cotadapter.ListenUDP(cfg, adapter)
}

func newCoTTCPListener(
	cfg cotadapter.TCPListenerConfig,
	adapter *cotadapter.Adapter,
) (cotTransport, error) {
	return cotadapter.ListenTCP(cfg, adapter)
}

func newNATSClient(cfg Config) (semstreamsClient, error) {
	return natsclient.NewClient(
		cfg.NATSURL,
		natsclient.WithName(cfg.NATSName),
		natsclient.WithTimeout(cfg.NATSConnectTimeout),
	)
}

func registerOwners(
	ctx context.Context,
	client semstreamsClient,
	heartbeatInterval time.Duration,
) (copownership.BindingResult, func(), error) {
	nats, ok := client.(*natsclient.Client)
	if !ok {
		return copownership.BindingResult{}, nil, fmt.Errorf("ownership registration requires *natsclient.Client")
	}
	registry, err := ownership.EnsureBuckets(ctx, nats, nil, nil)
	if err != nil {
		return copownership.BindingResult{}, nil, fmt.Errorf("ensure ownership buckets: %w", err)
	}
	heartbeater := registry.NewHeartbeater(heartbeatInterval)
	heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())
	go heartbeater.Run(heartbeatCtx)

	bindings, err := copownership.RegisterFirstPhase(ctx, registry, heartbeater)
	if err != nil {
		heartbeatCancel()
		return copownership.BindingResult{}, nil, err
	}
	return bindings, heartbeatCancel, nil
}
