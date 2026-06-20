package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	cotcomponent "github.com/c360studio/semops/internal/components/cot"
	mavcomponent "github.com/c360studio/semops/internal/components/mavlink"
	"github.com/c360studio/semops/internal/copownership"
	"github.com/c360studio/semops/internal/graphrequest"
	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
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
	cotUDPInput      *cotcomponent.UDPInputComponent
	cotTCPInput      *cotcomponent.TCPInputComponent
	cotDecoder       *cotcomponent.DecoderComponent
	cotProjector     *cotcomponent.ProjectorComponent
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
	newCoTPlanWriter     func(stack.CoTAdapterConfig, stack.CoTAdapterDeps) (cotcomponent.PlanWriter, error)
	newCoTUDPInput       func(cotcomponent.UDPInputConfig, cotcomponent.Bus) (*cotcomponent.UDPInputComponent, error)
	newCoTTCPInput       func(cotcomponent.TCPInputConfig, cotcomponent.Bus) (*cotcomponent.TCPInputComponent, error)
	newCoTDecoder        func(cotcomponent.DecoderConfig, cotcomponent.Bus) (*cotcomponent.DecoderComponent, error)
	newCoTProjector      func(cotcomponent.ProjectorConfig, cotcomponent.Bus) (*cotcomponent.ProjectorComponent, error)
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
	if a.mavlinkInput != nil {
		if err := a.mavlinkInput.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop MAVLink input component: %w", err))
		}
	}
	if a.cotUDPInput != nil {
		if err := a.cotUDPInput.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop CoT UDP input component: %w", err))
		}
	}
	if a.cotTCPInput != nil {
		if err := a.cotTCPInput.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop CoT TCP input component: %w", err))
		}
	}
	if a.mavlinkDecoder != nil {
		if err := a.mavlinkDecoder.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop MAVLink decoder component: %w", err))
		}
	}
	if a.cotDecoder != nil {
		if err := a.cotDecoder.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop CoT decoder component: %w", err))
		}
	}
	if a.mavlinkProjector != nil {
		if err := a.mavlinkProjector.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop MAVLink projector component: %w", err))
		}
	}
	if a.cotProjector != nil {
		if err := a.cotProjector.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop CoT projector component: %w", err))
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

func (a *App) CoTUDPInput() *cotcomponent.UDPInputComponent {
	if a == nil {
		return nil
	}
	return a.cotUDPInput
}

func (a *App) CoTTCPInput() *cotcomponent.TCPInputComponent {
	if a == nil {
		return nil
	}
	return a.cotTCPInput
}

func (a *App) CoTDecoder() *cotcomponent.DecoderComponent {
	if a == nil {
		return nil
	}
	return a.cotDecoder
}

func (a *App) CoTProjector() *cotcomponent.ProjectorComponent {
	if a == nil {
		return nil
	}
	return a.cotProjector
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
		if err := app.startCoTFlow(ctx, cfg.CoT, bindings, deps); err != nil {
			return nil, err
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
		newCoTPlanWriter:     newCoTPlanWriter,
		newCoTUDPInput:       cotcomponent.NewUDPInputComponent,
		newCoTTCPInput:       cotcomponent.NewTCPInputComponent,
		newCoTDecoder:        cotcomponent.NewDecoderComponent,
		newCoTProjector:      cotcomponent.NewProjectorComponent,
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
	if deps.newCoTPlanWriter == nil {
		deps.newCoTPlanWriter = defaults.newCoTPlanWriter
	}
	if deps.newCoTUDPInput == nil {
		deps.newCoTUDPInput = defaults.newCoTUDPInput
	}
	if deps.newCoTTCPInput == nil {
		deps.newCoTTCPInput = defaults.newCoTTCPInput
	}
	if deps.newCoTDecoder == nil {
		deps.newCoTDecoder = defaults.newCoTDecoder
	}
	if deps.newCoTProjector == nil {
		deps.newCoTProjector = defaults.newCoTProjector
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

func (a *App) startCoTFlow(
	ctx context.Context,
	cfg CoTConfig,
	bindings copownership.BindingResult,
	deps dependencies,
) error {
	bus := cotRuntimeBus{client: a.client}
	registry := payloadregistry.New()
	writer, err := deps.newCoTPlanWriter(stack.CoTAdapterConfig{
		Source:        cfg.Source,
		Org:           cfg.Org,
		Platform:      cfg.Platform,
		OwnerTokens:   bindings.OwnerTokenMap(),
		TraceID:       cfg.TraceID,
		RawMaxRecords: cfg.RawMaxRecords,
		RawMaxBytes:   cfg.RawMaxBytes,
		WriteTimeout:  cfg.WriteTimeout,
		Retry:         cfg.Retry,
	}, stack.CoTAdapterDeps{NATS: a.client})
	if err != nil {
		return fmt.Errorf("compose CoT graph writer: %w", err)
	}

	projector, err := deps.newCoTProjector(cotcomponent.ProjectorConfig{
		Registry: registry,
		Projector: cotprojector.NewProjector(cotprojector.Config{
			Org:         cfg.Org,
			Platform:    cfg.Platform,
			OwnerTokens: bindings.OwnerTokenMap(),
			TraceID:     cfg.TraceID,
		}),
		Writer:       writer,
		WriteTimeout: cfg.WriteTimeout,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose CoT projector component: %w", err)
	}
	decoder, err := deps.newCoTDecoder(cotcomponent.DecoderConfig{
		Source:         cfg.Source,
		RawSubject:     cotcomponent.DefaultRawSubject,
		DecodedSubject: cotcomponent.DefaultDecodedSubject,
		RawMaxRecords:  cfg.RawMaxRecords,
		RawMaxBytes:    cfg.RawMaxBytes,
		Registry:       registry,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose CoT decoder component: %w", err)
	}
	a.cotProjector = projector
	a.cotDecoder = decoder

	if err := startLifecycle(ctx, "CoT projector", projector); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "CoT decoder", decoder); err != nil {
		return err
	}
	if cfg.UDP.ListenAddr != "" {
		input, err := deps.newCoTUDPInput(cotcomponent.UDPInputConfig{
			Source:           cfg.Source,
			ListenAddr:       cfg.UDP.ListenAddr,
			RawSubject:       cotcomponent.DefaultRawSubject,
			MaxDatagramBytes: cfg.UDP.MaxDatagramBytes,
		}, bus)
		if err != nil {
			return fmt.Errorf("compose CoT UDP input component: %w", err)
		}
		a.cotUDPInput = input
		if err := startLifecycle(ctx, "CoT UDP input", input); err != nil {
			return err
		}
	}
	if cfg.TCP.ListenAddr != "" {
		input, err := deps.newCoTTCPInput(cotcomponent.TCPInputConfig{
			Source:        cfg.Source,
			ListenAddr:    cfg.TCP.ListenAddr,
			RawSubject:    cotcomponent.DefaultRawSubject,
			MaxEventBytes: cfg.TCP.MaxEventBytes,
		}, bus)
		if err != nil {
			return fmt.Errorf("compose CoT TCP input component: %w", err)
		}
		a.cotTCPInput = input
		if err := startLifecycle(ctx, "CoT TCP input", input); err != nil {
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

func newMAVLinkPlanWriter(
	cfg stack.MAVLinkAdapterConfig,
	deps stack.MAVLinkAdapterDeps,
) (mavcomponent.PlanWriter, error) {
	return stack.NewMAVLinkPlanWriter(cfg, deps)
}

func newCoTPlanWriter(
	cfg stack.CoTAdapterConfig,
	deps stack.CoTAdapterDeps,
) (cotcomponent.PlanWriter, error) {
	return stack.NewCoTPlanWriter(cfg, deps)
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

type cotRuntimeBus struct {
	client semstreamsClient
}

func (b cotRuntimeBus) Publish(ctx context.Context, subject string, data []byte) error {
	return b.client.Publish(ctx, subject, data)
}

func (b cotRuntimeBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (cotcomponent.Subscription, error) {
	subscription, err := b.client.Subscribe(ctx, subject, handler)
	if err != nil {
		return nil, err
	}
	if subscription == nil {
		return noopSubscription{}, nil
	}
	return subscription, nil
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
