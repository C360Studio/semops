package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	cotadapter "github.com/c360studio/semops/internal/adapters/cot"
	mavcomponent "github.com/c360studio/semops/internal/components/mavlink"
	"github.com/c360studio/semops/internal/copownership"
	"github.com/c360studio/semops/internal/graphrequest"
	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	"github.com/c360studio/semops/internal/stack"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/nats-io/nats.go"
)

type App struct {
	client           semstreamsClient
	ownershipStop    func()
	ownershipResult  copownership.BindingResult
	mavlinkInput     *mavcomponent.UDPInputComponent
	mavlinkDecoder   *mavcomponent.DecoderComponent
	mavlinkProjector *mavcomponent.ProjectorComponent
	cotAdapter       *cotadapter.Adapter
	cotTransports    []runningCoTTransport
}

type semstreamsClient interface {
	graphrequest.RetryRequester
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	RequestClassified(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	Publish(ctx context.Context, subject string, data []byte) error
	Subscribe(ctx context.Context, subject string, handler func(context.Context, *nats.Msg)) (*natsclient.Subscription, error)
	Connect(context.Context) error
	Close(context.Context) error
}

type GraphRequester interface {
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	RequestClassified(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
}

type dependencies struct {
	newNATSClient        func(Config) (semstreamsClient, error)
	registerOwners       func(context.Context, semstreamsClient, time.Duration) (copownership.BindingResult, func(), error)
	newMAVLinkPlanWriter func(stack.MAVLinkAdapterConfig, stack.MAVLinkAdapterDeps) (mavcomponent.PlanWriter, error)
	newMAVLinkUDPInput   func(mavcomponent.UDPInputConfig, mavcomponent.Bus) (*mavcomponent.UDPInputComponent, error)
	newMAVLinkDecoder    func(mavcomponent.DecoderConfig, mavcomponent.Bus) (*mavcomponent.DecoderComponent, error)
	newMAVLinkProjector  func(mavcomponent.ProjectorConfig, mavcomponent.Bus) (*mavcomponent.ProjectorComponent, error)
	newCoTAdapter        func(stack.CoTAdapterConfig, stack.CoTAdapterDeps) (*cotadapter.Adapter, error)
	newCoTUDPListener    func(cotadapter.UDPListenerConfig, *cotadapter.Adapter) (cotTransport, error)
	newCoTTCPListener    func(cotadapter.TCPListenerConfig, *cotadapter.Adapter) (cotTransport, error)
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
	for _, running := range a.cotTransports {
		if running.cancel != nil {
			running.cancel()
		}
	}
	if a.mavlinkInput != nil {
		if err := a.mavlinkInput.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop MAVLink input component: %w", err))
		}
	}
	if a.mavlinkDecoder != nil {
		if err := a.mavlinkDecoder.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop MAVLink decoder component: %w", err))
		}
	}
	if a.mavlinkProjector != nil {
		if err := a.mavlinkProjector.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop MAVLink projector component: %w", err))
		}
	}
	for _, running := range a.cotTransports {
		if running.transport != nil {
			if err := running.transport.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close %s transport: %w", running.name, err))
			}
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

func (a *App) MAVLinkInput() *mavcomponent.UDPInputComponent {
	if a == nil {
		return nil
	}
	return a.mavlinkInput
}

func (a *App) MAVLinkDecoder() *mavcomponent.DecoderComponent {
	if a == nil {
		return nil
	}
	return a.mavlinkDecoder
}

func (a *App) MAVLinkProjector() *mavcomponent.ProjectorComponent {
	if a == nil {
		return nil
	}
	return a.mavlinkProjector
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
		if err := app.startMAVLinkFlow(ctx, cfg.MAVLink, bindings, deps); err != nil {
			return nil, err
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
		newNATSClient:        newNATSClient,
		registerOwners:       registerOwners,
		newMAVLinkPlanWriter: newMAVLinkPlanWriter,
		newMAVLinkUDPInput:   mavcomponent.NewUDPInputComponent,
		newMAVLinkDecoder:    mavcomponent.NewDecoderComponent,
		newMAVLinkProjector:  mavcomponent.NewProjectorComponent,
		newCoTAdapter:        stack.NewCoTAdapter,
		newCoTUDPListener:    newCoTUDPListener,
		newCoTTCPListener:    newCoTTCPListener,
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
	if deps.newMAVLinkPlanWriter == nil {
		deps.newMAVLinkPlanWriter = defaults.newMAVLinkPlanWriter
	}
	if deps.newMAVLinkUDPInput == nil {
		deps.newMAVLinkUDPInput = defaults.newMAVLinkUDPInput
	}
	if deps.newMAVLinkDecoder == nil {
		deps.newMAVLinkDecoder = defaults.newMAVLinkDecoder
	}
	if deps.newMAVLinkProjector == nil {
		deps.newMAVLinkProjector = defaults.newMAVLinkProjector
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

func (a *App) startMAVLinkFlow(
	ctx context.Context,
	cfg MAVLinkConfig,
	bindings copownership.BindingResult,
	deps dependencies,
) error {
	bus := runtimeBus{client: a.client}
	registry := payloadregistry.New()
	writer, err := deps.newMAVLinkPlanWriter(stack.MAVLinkAdapterConfig{
		Source:        cfg.Source,
		Org:           cfg.Org,
		Platform:      cfg.Platform,
		OwnerTokens:   bindings.OwnerTokenMap(),
		TraceID:       cfg.TraceID,
		RawMaxRecords: cfg.RawMaxRecords,
		RawMaxBytes:   cfg.RawMaxBytes,
		WriteTimeout:  cfg.WriteTimeout,
		Retry:         cfg.Retry,
	}, stack.MAVLinkAdapterDeps{NATS: a.client})
	if err != nil {
		return fmt.Errorf("compose MAVLink graph writer: %w", err)
	}

	projector, err := deps.newMAVLinkProjector(mavcomponent.ProjectorConfig{
		Registry: registry,
		Projector: mavprojector.NewProjector(mavprojector.Config{
			Org:         cfg.Org,
			Platform:    cfg.Platform,
			OwnerTokens: bindings.OwnerTokenMap(),
			TraceID:     cfg.TraceID,
		}),
		Writer:       writer,
		WriteTimeout: cfg.WriteTimeout,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose MAVLink projector component: %w", err)
	}
	decoder, err := deps.newMAVLinkDecoder(mavcomponent.DecoderConfig{
		Source:         cfg.Source,
		RawSubject:     mavcomponent.DefaultRawSubject,
		DecodedSubject: mavcomponent.DefaultDecodedSubject,
		RawMaxRecords:  cfg.RawMaxRecords,
		RawMaxBytes:    cfg.RawMaxBytes,
		Registry:       registry,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose MAVLink decoder component: %w", err)
	}
	a.mavlinkProjector = projector
	a.mavlinkDecoder = decoder

	if err := startLifecycle(ctx, "MAVLink projector", projector); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "MAVLink decoder", decoder); err != nil {
		return err
	}
	if cfg.UDP.ListenAddr != "" {
		input, err := deps.newMAVLinkUDPInput(mavcomponent.UDPInputConfig{
			Source:           cfg.Source,
			ListenAddr:       cfg.UDP.ListenAddr,
			RawSubject:       mavcomponent.DefaultRawSubject,
			MaxDatagramBytes: cfg.UDP.MaxDatagramBytes,
		}, bus)
		if err != nil {
			return fmt.Errorf("compose MAVLink UDP input component: %w", err)
		}
		a.mavlinkInput = input
		if err := startLifecycle(ctx, "MAVLink UDP input", input); err != nil {
			return err
		}
	}
	return nil
}

func startLifecycle(ctx context.Context, name string, lifecycle interface {
	Initialize() error
	Start(context.Context) error
}) error {
	if err := lifecycle.Initialize(); err != nil {
		return fmt.Errorf("initialize %s: %w", name, err)
	}
	if err := lifecycle.Start(ctx); err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}
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

func newMAVLinkPlanWriter(
	cfg stack.MAVLinkAdapterConfig,
	deps stack.MAVLinkAdapterDeps,
) (mavcomponent.PlanWriter, error) {
	return stack.NewMAVLinkPlanWriter(cfg, deps)
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

type runtimeBus struct {
	client semstreamsClient
}

func (b runtimeBus) Publish(ctx context.Context, subject string, data []byte) error {
	return b.client.Publish(ctx, subject, data)
}

func (b runtimeBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (mavcomponent.Subscription, error) {
	subscription, err := b.client.Subscribe(ctx, subject, handler)
	if err != nil {
		return nil, err
	}
	if subscription == nil {
		return noopSubscription{}, nil
	}
	return subscription, nil
}

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() error {
	return nil
}

func remainingTimeout(ctx context.Context) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			return remaining
		}
		return time.Nanosecond
	}
	return time.Second
}
