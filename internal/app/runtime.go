package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	adsbcomponent "github.com/c360studio/semops/internal/components/adsb"
	capcomponent "github.com/c360studio/semops/internal/components/cap"
	cotcomponent "github.com/c360studio/semops/internal/components/cot"
	mavcomponent "github.com/c360studio/semops/internal/components/mavlink"
	sapientcomponent "github.com/c360studio/semops/internal/components/sapient"
	"github.com/c360studio/semops/internal/copownership"
	"github.com/c360studio/semops/internal/graphrequest"
	adsbprojector "github.com/c360studio/semops/internal/projectors/adsb"
	capprojector "github.com/c360studio/semops/internal/projectors/cap"
	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	"github.com/c360studio/semops/internal/stack"
	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	capcodec "github.com/c360studio/semops/pkg/adapters/cap"
	sapientcodec "github.com/c360studio/semops/pkg/adapters/sapient"
	"github.com/c360studio/semops/pkg/cop"
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
	capPoller        *capcomponent.HTTPPollerComponent
	capDecoder       *capcomponent.DecoderComponent
	capProjector     *capcomponent.ProjectorComponent
	adsbPoller       *adsbcomponent.HTTPPollerComponent
	adsbDecoder      *adsbcomponent.DecoderComponent
	adsbProjector    *adsbcomponent.ProjectorComponent
	sapientInput     *sapientcomponent.HTTPInputComponent
	sapientDecoder   *sapientcomponent.DecoderComponent
	runtimeCancel    context.CancelFunc
}

type semstreamsClient interface {
	graphrequest.RetryRequester
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	RequestClassified(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	Publish(ctx context.Context, subject string, data []byte) error
	Subscribe(
		ctx context.Context,
		subject string,
		handler func(context.Context, *nats.Msg),
	) (*natsclient.Subscription, error)
	Connect(context.Context) error
	Close(context.Context) error
}

type GraphRequester interface {
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	RequestClassified(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
}

type dependencies struct {
	newNATSClient        func(Config) (semstreamsClient, error)
	registerOwners       func(context.Context, semstreamsClient, time.Duration, []cop.OwnedContract) (copownership.BindingResult, func(), error)
	newMAVLinkPlanWriter func(stack.MAVLinkAdapterConfig, stack.MAVLinkAdapterDeps) (mavcomponent.PlanWriter, error)
	newMAVLinkUDPInput   func(mavcomponent.UDPInputConfig, mavcomponent.Bus) (*mavcomponent.UDPInputComponent, error)
	newMAVLinkDecoder    func(mavcomponent.DecoderConfig, mavcomponent.Bus) (*mavcomponent.DecoderComponent, error)
	newMAVLinkProjector  func(mavcomponent.ProjectorConfig, mavcomponent.Bus) (*mavcomponent.ProjectorComponent, error)
	newCoTPlanWriter     func(stack.CoTAdapterConfig, stack.CoTAdapterDeps) (cotcomponent.PlanWriter, error)
	newCoTUDPInput       func(cotcomponent.UDPInputConfig, cotcomponent.Bus) (*cotcomponent.UDPInputComponent, error)
	newCoTTCPInput       func(cotcomponent.TCPInputConfig, cotcomponent.Bus) (*cotcomponent.TCPInputComponent, error)
	newCoTDecoder        func(cotcomponent.DecoderConfig, cotcomponent.Bus) (*cotcomponent.DecoderComponent, error)
	newCoTProjector      func(cotcomponent.ProjectorConfig, cotcomponent.Bus) (*cotcomponent.ProjectorComponent, error)
	newCAPPlanWriter     func(stack.CAPAdapterConfig, stack.CAPAdapterDeps) (capcomponent.PlanWriter, error)
	newCAPHTTPPoller     func(capcomponent.HTTPPollerConfig, capcomponent.Bus) (*capcomponent.HTTPPollerComponent, error)
	newCAPDecoder        func(capcomponent.DecoderConfig, capcomponent.Bus) (*capcomponent.DecoderComponent, error)
	newCAPProjector      func(capcomponent.ProjectorConfig, capcomponent.Bus) (*capcomponent.ProjectorComponent, error)
	newADSBPlanWriter    func(stack.ADSBAdapterConfig, stack.ADSBAdapterDeps) (adsbcomponent.PlanWriter, error)
	newADSBHTTPPoller    func(adsbcomponent.HTTPPollerConfig, adsbcomponent.Bus) (*adsbcomponent.HTTPPollerComponent, error)
	newADSBDecoder       func(adsbcomponent.DecoderConfig, adsbcomponent.Bus) (*adsbcomponent.DecoderComponent, error)
	newADSBProjector     func(adsbcomponent.ProjectorConfig, adsbcomponent.Bus) (*adsbcomponent.ProjectorComponent, error)
	newSAPIENTHTTPInput  func(sapientcomponent.HTTPInputConfig, sapientcomponent.Bus) (*sapientcomponent.HTTPInputComponent, error)
	newSAPIENTDecoder    func(sapientcomponent.DecoderConfig, sapientcomponent.Bus) (*sapientcomponent.DecoderComponent, error)
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
	if a.runtimeCancel != nil {
		a.runtimeCancel()
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
	if a.capPoller != nil {
		if err := a.capPoller.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop CAP HTTP poller component: %w", err))
		}
	}
	if a.adsbPoller != nil {
		if err := a.adsbPoller.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop ADS-B HTTP poller component: %w", err))
		}
	}
	if a.sapientInput != nil {
		if err := a.sapientInput.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop SAPIENT HTTP input component: %w", err))
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
	if a.capDecoder != nil {
		if err := a.capDecoder.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop CAP decoder component: %w", err))
		}
	}
	if a.adsbDecoder != nil {
		if err := a.adsbDecoder.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop ADS-B decoder component: %w", err))
		}
	}
	if a.sapientDecoder != nil {
		if err := a.sapientDecoder.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop SAPIENT decoder component: %w", err))
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
	if a.capProjector != nil {
		if err := a.capProjector.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop CAP projector component: %w", err))
		}
	}
	if a.adsbProjector != nil {
		if err := a.adsbProjector.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop ADS-B projector component: %w", err))
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

func (a *App) CAPHTTPPoller() *capcomponent.HTTPPollerComponent {
	if a == nil {
		return nil
	}
	return a.capPoller
}

func (a *App) CAPDecoder() *capcomponent.DecoderComponent {
	if a == nil {
		return nil
	}
	return a.capDecoder
}

func (a *App) CAPProjector() *capcomponent.ProjectorComponent {
	if a == nil {
		return nil
	}
	return a.capProjector
}

func (a *App) ADSBHTTPPoller() *adsbcomponent.HTTPPollerComponent {
	if a == nil {
		return nil
	}
	return a.adsbPoller
}

func (a *App) ADSBDecoder() *adsbcomponent.DecoderComponent {
	if a == nil {
		return nil
	}
	return a.adsbDecoder
}

func (a *App) ADSBProjector() *adsbcomponent.ProjectorComponent {
	if a == nil {
		return nil
	}
	return a.adsbProjector
}

func (a *App) SAPIENTHTTPInput() *sapientcomponent.HTTPInputComponent {
	if a == nil {
		return nil
	}
	return a.sapientInput
}

func (a *App) SAPIENTDecoder() *sapientcomponent.DecoderComponent {
	if a == nil {
		return nil
	}
	return a.sapientDecoder
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

	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	app := &App{client: client, runtimeCancel: runtimeCancel}
	cleanup := true
	defer func() {
		if cleanup {
			_ = app.Close(context.Background())
		}
	}()

	bindings, stopOwners, err := deps.registerOwners(
		ctx,
		client,
		cfg.OwnershipHeartbeatInterval,
		runtimeOwnedContracts(cfg),
	)
	if err != nil {
		return nil, fmt.Errorf("register SemOps COP ownership: %w", err)
	}
	app.ownershipResult = bindings
	app.ownershipStop = stopOwners

	if cfg.MAVLink.Enabled {
		if err := app.startMAVLinkFlow(runtimeCtx, cfg.MAVLink, bindings, deps); err != nil {
			return nil, err
		}
	}

	if cfg.CoT.Enabled {
		if err := app.startCoTFlow(runtimeCtx, cfg.CoT, bindings, deps); err != nil {
			return nil, err
		}
	}

	if cfg.CAP.Enabled {
		if err := app.startCAPFlow(runtimeCtx, cfg.CAP, bindings, deps); err != nil {
			return nil, err
		}
	}

	if cfg.ADSB.Enabled {
		if err := app.startADSBFlow(runtimeCtx, cfg.ADSB, bindings, deps); err != nil {
			return nil, err
		}
	}

	if cfg.SAPIENT.Enabled {
		if err := app.startSAPIENTFlow(runtimeCtx, cfg.SAPIENT, deps); err != nil {
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
		newCAPPlanWriter:     newCAPPlanWriter,
		newCAPHTTPPoller:     capcomponent.NewHTTPPollerComponent,
		newCAPDecoder:        capcomponent.NewDecoderComponent,
		newCAPProjector:      capcomponent.NewProjectorComponent,
		newADSBPlanWriter:    newADSBPlanWriter,
		newADSBHTTPPoller:    adsbcomponent.NewHTTPPollerComponent,
		newADSBDecoder:       adsbcomponent.NewDecoderComponent,
		newADSBProjector:     adsbcomponent.NewProjectorComponent,
		newSAPIENTHTTPInput:  sapientcomponent.NewHTTPInputComponent,
		newSAPIENTDecoder:    sapientcomponent.NewDecoderComponent,
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
	if deps.newCAPPlanWriter == nil {
		deps.newCAPPlanWriter = defaults.newCAPPlanWriter
	}
	if deps.newCAPHTTPPoller == nil {
		deps.newCAPHTTPPoller = defaults.newCAPHTTPPoller
	}
	if deps.newCAPDecoder == nil {
		deps.newCAPDecoder = defaults.newCAPDecoder
	}
	if deps.newCAPProjector == nil {
		deps.newCAPProjector = defaults.newCAPProjector
	}
	if deps.newADSBPlanWriter == nil {
		deps.newADSBPlanWriter = defaults.newADSBPlanWriter
	}
	if deps.newADSBHTTPPoller == nil {
		deps.newADSBHTTPPoller = defaults.newADSBHTTPPoller
	}
	if deps.newADSBDecoder == nil {
		deps.newADSBDecoder = defaults.newADSBDecoder
	}
	if deps.newADSBProjector == nil {
		deps.newADSBProjector = defaults.newADSBProjector
	}
	if deps.newSAPIENTHTTPInput == nil {
		deps.newSAPIENTHTTPInput = defaults.newSAPIENTHTTPInput
	}
	if deps.newSAPIENTDecoder == nil {
		deps.newSAPIENTDecoder = defaults.newSAPIENTDecoder
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

func (a *App) startCAPFlow(
	ctx context.Context,
	cfg CAPConfig,
	bindings copownership.BindingResult,
	deps dependencies,
) error {
	bus := capRuntimeBus{client: a.client}
	registry := payloadregistry.New()
	writer, err := deps.newCAPPlanWriter(stack.CAPAdapterConfig{
		Source:       cfg.Source,
		Org:          cfg.Org,
		Platform:     cfg.Platform,
		OwnerTokens:  bindings.OwnerTokenMap(),
		TraceID:      cfg.TraceID,
		WriteTimeout: cfg.WriteTimeout,
		Retry:        cfg.Retry,
	}, stack.CAPAdapterDeps{NATS: a.client})
	if err != nil {
		return fmt.Errorf("compose CAP graph writer: %w", err)
	}

	projector, err := deps.newCAPProjector(capcomponent.ProjectorConfig{
		Registry: registry,
		Projector: capprojector.NewProjector(capprojector.Config{
			Org:         cfg.Org,
			Platform:    cfg.Platform,
			OwnerTokens: bindings.OwnerTokenMap(),
			TraceID:     cfg.TraceID,
		}),
		Writer:       writer,
		WriteTimeout: cfg.WriteTimeout,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose CAP projector component: %w", err)
	}
	var replay capcomponent.ReplayAppender
	if cfg.ReplayPath != "" {
		replay = capcodec.NewReplayStore(cfg.ReplayPath)
	}
	decoder, err := deps.newCAPDecoder(capcomponent.DecoderConfig{
		Source:         cfg.Source,
		RawSubject:     capcomponent.DefaultRawSubject,
		DecodedSubject: capcomponent.DefaultDecodedSubject,
		Registry:       registry,
		Replay:         replay,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose CAP decoder component: %w", err)
	}
	poller, err := deps.newCAPHTTPPoller(capcomponent.HTTPPollerConfig{
		Source:           cfg.Source,
		URL:              cfg.HTTP.URL,
		Method:           cfg.HTTP.Method,
		RawSubject:       capcomponent.DefaultRawSubject,
		PollInterval:     cfg.HTTP.PollInterval,
		StaleAfter:       cfg.HTTP.StaleAfter,
		ContactPolicy:    cfg.HTTP.ContactPolicy,
		AuthRef:          cfg.HTTP.AuthRef,
		MaxResponseBytes: cfg.HTTP.MaxResponseBytes,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose CAP HTTP poller component: %w", err)
	}
	a.capProjector = projector
	a.capDecoder = decoder
	a.capPoller = poller

	if err := startLifecycle(ctx, "CAP projector", projector); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "CAP decoder", decoder); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "CAP HTTP poller", poller); err != nil {
		return err
	}
	return nil
}

func (a *App) startADSBFlow(
	ctx context.Context,
	cfg ADSBConfig,
	bindings copownership.BindingResult,
	deps dependencies,
) error {
	bus := adsbRuntimeBus{client: a.client}
	registry := payloadregistry.New()
	writer, err := deps.newADSBPlanWriter(stack.ADSBAdapterConfig{
		Source:        cfg.Source,
		Org:           cfg.Org,
		Platform:      cfg.Platform,
		OwnerTokens:   bindings.OwnerTokenMap(),
		TraceID:       cfg.TraceID,
		RawMaxRecords: cfg.RawMaxRecords,
		RawMaxBytes:   cfg.RawMaxBytes,
		WriteTimeout:  cfg.WriteTimeout,
		Retry:         cfg.Retry,
	}, stack.ADSBAdapterDeps{NATS: a.client})
	if err != nil {
		return fmt.Errorf("compose ADS-B graph writer: %w", err)
	}

	projector, err := deps.newADSBProjector(adsbcomponent.ProjectorConfig{
		Registry: registry,
		Projector: adsbprojector.NewProjector(adsbprojector.Config{
			Org:         cfg.Org,
			Platform:    cfg.Platform,
			OwnerTokens: bindings.OwnerTokenMap(),
			TraceID:     cfg.TraceID,
		}),
		Writer:       writer,
		WriteTimeout: cfg.WriteTimeout,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose ADS-B projector component: %w", err)
	}
	var replay adsbcomponent.ReplayAppender
	if cfg.ReplayPath != "" {
		replay = adsbcodec.NewReplayStore(cfg.ReplayPath)
	}
	decoder, err := deps.newADSBDecoder(adsbcomponent.DecoderConfig{
		Source:         cfg.Source,
		RawSubject:     adsbcomponent.DefaultRawSubject,
		DecodedSubject: adsbcomponent.DefaultDecodedSubject,
		RawMaxRecords:  cfg.RawMaxRecords,
		RawMaxBytes:    cfg.RawMaxBytes,
		Registry:       registry,
		Replay:         replay,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose ADS-B decoder component: %w", err)
	}
	poller, err := deps.newADSBHTTPPoller(adsbcomponent.HTTPPollerConfig{
		Source:           cfg.Source,
		URL:              cfg.HTTP.URL,
		Method:           cfg.HTTP.Method,
		RawSubject:       adsbcomponent.DefaultRawSubject,
		PollInterval:     cfg.HTTP.PollInterval,
		StaleAfter:       cfg.HTTP.StaleAfter,
		ContactPolicy:    cfg.HTTP.ContactPolicy,
		AuthRef:          cfg.HTTP.AuthRef,
		MaxResponseBytes: cfg.HTTP.MaxResponseBytes,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose ADS-B HTTP poller component: %w", err)
	}
	a.adsbProjector = projector
	a.adsbDecoder = decoder
	a.adsbPoller = poller

	if err := startLifecycle(ctx, "ADS-B projector", projector); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "ADS-B decoder", decoder); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "ADS-B HTTP poller", poller); err != nil {
		return err
	}
	return nil
}

func (a *App) startSAPIENTFlow(ctx context.Context, cfg SAPIENTConfig, deps dependencies) error {
	bus := sapientRuntimeBus{client: a.client}
	registry := payloadregistry.New()
	var replay sapientcomponent.ReplayAppender
	if cfg.ReplayPath != "" {
		replay = sapientcodec.NewReplayStore(cfg.ReplayPath)
	}
	decoder, err := deps.newSAPIENTDecoder(sapientcomponent.DecoderConfig{
		Source:         cfg.Source,
		RawSubject:     sapientcomponent.DefaultRawSubject,
		DecodedSubject: sapientcomponent.DefaultDecodedSubject,
		RawMaxRecords:  cfg.RawMaxRecords,
		RawMaxBytes:    cfg.RawMaxBytes,
		Registry:       registry,
		Replay:         replay,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose SAPIENT decoder component: %w", err)
	}
	input, err := deps.newSAPIENTHTTPInput(sapientcomponent.HTTPInputConfig{
		Source:           cfg.Source,
		URL:              cfg.HTTP.URL,
		Method:           cfg.HTTP.Method,
		RawSubject:       sapientcomponent.DefaultRawSubject,
		PollInterval:     cfg.HTTP.PollInterval,
		StaleAfter:       cfg.HTTP.StaleAfter,
		ContactPolicy:    cfg.HTTP.ContactPolicy,
		AuthRef:          cfg.HTTP.AuthRef,
		MaxResponseBytes: cfg.HTTP.MaxResponseBytes,
		Encoding:         cfg.HTTP.Encoding,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose SAPIENT HTTP input component: %w", err)
	}
	a.sapientDecoder = decoder
	a.sapientInput = input

	if err := startLifecycle(ctx, "SAPIENT decoder", decoder); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "SAPIENT HTTP input", input); err != nil {
		return err
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

func newCAPPlanWriter(
	cfg stack.CAPAdapterConfig,
	deps stack.CAPAdapterDeps,
) (capcomponent.PlanWriter, error) {
	return stack.NewCAPPlanWriter(cfg, deps)
}

func newADSBPlanWriter(
	cfg stack.ADSBAdapterConfig,
	deps stack.ADSBAdapterDeps,
) (adsbcomponent.PlanWriter, error) {
	return stack.NewADSBPlanWriter(cfg, deps)
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
	owned []cop.OwnedContract,
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

	bindings, err := copownership.RegisterOwnedContracts(ctx, registry, heartbeater, owned)
	if err != nil {
		heartbeatCancel()
		return copownership.BindingResult{}, nil, err
	}
	return bindings, heartbeatCancel, nil
}

func runtimeOwnedContracts(cfg Config) []cop.OwnedContract {
	owned := cop.FirstPhaseOwnedContracts()
	if cfg.ADSB.Enabled {
		owned = append(owned, cop.OwnedContract{
			Owner:    cop.OwnerADSB,
			Contract: cop.ADSBTrackContract(),
		})
	}
	return owned
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

type capRuntimeBus struct {
	client semstreamsClient
}

func (b capRuntimeBus) Publish(ctx context.Context, subject string, data []byte) error {
	return b.client.Publish(ctx, subject, data)
}

func (b capRuntimeBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (capcomponent.Subscription, error) {
	subscription, err := b.client.Subscribe(ctx, subject, handler)
	if err != nil {
		return nil, err
	}
	if subscription == nil {
		return noopSubscription{}, nil
	}
	return subscription, nil
}

type adsbRuntimeBus struct {
	client semstreamsClient
}

func (b adsbRuntimeBus) Publish(ctx context.Context, subject string, data []byte) error {
	return b.client.Publish(ctx, subject, data)
}

func (b adsbRuntimeBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (adsbcomponent.Subscription, error) {
	subscription, err := b.client.Subscribe(ctx, subject, handler)
	if err != nil {
		return nil, err
	}
	if subscription == nil {
		return noopSubscription{}, nil
	}
	return subscription, nil
}

type sapientRuntimeBus struct {
	client semstreamsClient
}

func (b sapientRuntimeBus) Publish(ctx context.Context, subject string, data []byte) error {
	return b.client.Publish(ctx, subject, data)
}

func (b sapientRuntimeBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (sapientcomponent.Subscription, error) {
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
