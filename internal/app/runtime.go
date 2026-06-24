package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	sapientProjector *sapientcomponent.ProjectorComponent
	klvMediaInput    *klvcomponent.MediaRefInputComponent
	klvDemux         *klvcomponent.DemuxComponent
	klvDecoder       *klvcomponent.DecoderComponent
	klvProjector     *klvcomponent.ProjectorComponent
	weatherInput     *weathercomponent.FixtureInputComponent
	weatherDecoder   *weathercomponent.DecoderComponent
	weatherProjector *weathercomponent.ProjectorComponent
	fusionCandidates *fusioncomponent.CandidateProducerComponent
	fusionProjector  *fusioncomponent.ProjectorComponent
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
	newSAPIENTPlanWriter func(time.Duration, natsclient.RetryConfig, graphrequest.RetryRequester) (sapientcomponent.PlanWriter, error)
	newSAPIENTHTTPInput  func(sapientcomponent.HTTPInputConfig, sapientcomponent.Bus) (*sapientcomponent.HTTPInputComponent, error)
	newSAPIENTDecoder    func(sapientcomponent.DecoderConfig, sapientcomponent.Bus) (*sapientcomponent.DecoderComponent, error)
	newSAPIENTProjector  func(sapientcomponent.ProjectorConfig, sapientcomponent.Bus) (*sapientcomponent.ProjectorComponent, error)
	newKLVPlanWriter     func(stack.KLVAdapterConfig, stack.KLVAdapterDeps) (klvcomponent.ProjectorPlanWriter, error)
	newKLVMediaRefInput  func(klvcomponent.MediaRefInputConfig) (*klvcomponent.MediaRefInputComponent, error)
	newKLVDemux          func(klvcomponent.DemuxConfig) (*klvcomponent.DemuxComponent, error)
	newKLVDecoder        func(klvcomponent.DecoderConfig) (*klvcomponent.DecoderComponent, error)
	newKLVProjector      func(klvcomponent.ProjectorConfig) (*klvcomponent.ProjectorComponent, error)
	newWeatherPlanWriter func(stack.WeatherAdapterConfig, stack.WeatherAdapterDeps) (weathercomponent.PlanWriter, error)
	newWeatherInput      func(weathercomponent.FixtureInputConfig) (*weathercomponent.FixtureInputComponent, error)
	newWeatherDecoder    func(weathercomponent.DecoderConfig, weathercomponent.Bus) (*weathercomponent.DecoderComponent, error)
	newWeatherProjector  func(weathercomponent.ProjectorConfig, weathercomponent.Bus) (*weathercomponent.ProjectorComponent, error)
	newFusionCandidates  func(fusioncomponent.CandidateProducerConfig, fusioncomponent.Bus) (*fusioncomponent.CandidateProducerComponent, error)
	newFusionPlanWriter  func(time.Duration, natsclient.RetryConfig, graphrequest.RetryRequester) (fusioncomponent.PlanWriter, error)
	newFusionProjector   func(fusioncomponent.ProjectorConfig, fusioncomponent.Bus) (*fusioncomponent.ProjectorComponent, error)
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
	if a.klvMediaInput != nil {
		if err := a.klvMediaInput.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop KLV media-ref input component: %w", err))
		}
	}
	if a.weatherInput != nil {
		if err := a.weatherInput.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop weather fixture input component: %w", err))
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
	if a.klvDemux != nil {
		if err := a.klvDemux.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop KLV demux component: %w", err))
		}
	}
	if a.klvDecoder != nil {
		if err := a.klvDecoder.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop KLV decoder component: %w", err))
		}
	}
	if a.weatherDecoder != nil {
		if err := a.weatherDecoder.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop weather decoder component: %w", err))
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
	if a.sapientProjector != nil {
		if err := a.sapientProjector.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop SAPIENT projector component: %w", err))
		}
	}
	if a.klvProjector != nil {
		if err := a.klvProjector.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop KLV projector component: %w", err))
		}
	}
	if a.weatherProjector != nil {
		if err := a.weatherProjector.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop weather projector component: %w", err))
		}
	}
	if a.fusionCandidates != nil {
		if err := a.fusionCandidates.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop fusion candidate producer component: %w", err))
		}
	}
	if a.fusionProjector != nil {
		if err := a.fusionProjector.Stop(remainingTimeout(ctx)); err != nil {
			errs = append(errs, fmt.Errorf("stop fusion projector component: %w", err))
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

func (a *App) SAPIENTProjector() *sapientcomponent.ProjectorComponent {
	if a == nil {
		return nil
	}
	return a.sapientProjector
}

func (a *App) KLVMediaRefInput() *klvcomponent.MediaRefInputComponent {
	if a == nil {
		return nil
	}
	return a.klvMediaInput
}

func (a *App) KLVDemux() *klvcomponent.DemuxComponent {
	if a == nil {
		return nil
	}
	return a.klvDemux
}

func (a *App) KLVDecoder() *klvcomponent.DecoderComponent {
	if a == nil {
		return nil
	}
	return a.klvDecoder
}

func (a *App) KLVProjector() *klvcomponent.ProjectorComponent {
	if a == nil {
		return nil
	}
	return a.klvProjector
}

func (a *App) WeatherInput() *weathercomponent.FixtureInputComponent {
	if a == nil {
		return nil
	}
	return a.weatherInput
}

func (a *App) WeatherDecoder() *weathercomponent.DecoderComponent {
	if a == nil {
		return nil
	}
	return a.weatherDecoder
}

func (a *App) WeatherProjector() *weathercomponent.ProjectorComponent {
	if a == nil {
		return nil
	}
	return a.weatherProjector
}

func (a *App) FusionProjector() *fusioncomponent.ProjectorComponent {
	if a == nil {
		return nil
	}
	return a.fusionProjector
}

func (a *App) FusionCandidateProducer() *fusioncomponent.CandidateProducerComponent {
	if a == nil {
		return nil
	}
	return a.fusionCandidates
}

func (a *App) GraphRequester() GraphRequester {
	if a == nil {
		return nil
	}
	return a.client
}

func (a *App) ComponentMetricSources() []componentmetrics.Source {
	if a == nil {
		return nil
	}
	sources := make([]componentmetrics.Source, 0, 24)
	if a.mavlinkInput != nil {
		sources = append(sources, componentmetrics.Source{Feed: "mavlink", Role: "input", Component: a.mavlinkInput})
	}
	if a.mavlinkDecoder != nil {
		sources = append(sources, componentmetrics.Source{Feed: "mavlink", Role: "decoder", Component: a.mavlinkDecoder})
	}
	if a.mavlinkProjector != nil {
		sources = append(sources, componentmetrics.Source{Feed: "mavlink", Role: "projector", Component: a.mavlinkProjector})
	}
	if a.cotUDPInput != nil {
		sources = append(sources, componentmetrics.Source{Feed: "tak-cot", Role: "udp-input", Component: a.cotUDPInput})
	}
	if a.cotTCPInput != nil {
		sources = append(sources, componentmetrics.Source{Feed: "tak-cot", Role: "tcp-input", Component: a.cotTCPInput})
	}
	if a.cotDecoder != nil {
		sources = append(sources, componentmetrics.Source{Feed: "tak-cot", Role: "decoder", Component: a.cotDecoder})
	}
	if a.cotProjector != nil {
		sources = append(sources, componentmetrics.Source{Feed: "tak-cot", Role: "projector", Component: a.cotProjector})
	}
	if a.capPoller != nil {
		sources = append(sources, componentmetrics.Source{Feed: "cap", Role: "http-poller", Component: a.capPoller})
	}
	if a.capDecoder != nil {
		sources = append(sources, componentmetrics.Source{Feed: "cap", Role: "decoder", Component: a.capDecoder})
	}
	if a.capProjector != nil {
		sources = append(sources, componentmetrics.Source{Feed: "cap", Role: "projector", Component: a.capProjector})
	}
	if a.adsbPoller != nil {
		sources = append(sources, componentmetrics.Source{Feed: "adsb", Role: "http-poller", Component: a.adsbPoller})
	}
	if a.adsbDecoder != nil {
		sources = append(sources, componentmetrics.Source{Feed: "adsb", Role: "decoder", Component: a.adsbDecoder})
	}
	if a.adsbProjector != nil {
		sources = append(sources, componentmetrics.Source{Feed: "adsb", Role: "projector", Component: a.adsbProjector})
	}
	if a.sapientInput != nil {
		sources = append(sources, componentmetrics.Source{Feed: "sapient", Role: "http-input", Component: a.sapientInput})
	}
	if a.sapientDecoder != nil {
		sources = append(sources, componentmetrics.Source{Feed: "sapient", Role: "decoder", Component: a.sapientDecoder})
	}
	if a.sapientProjector != nil {
		sources = append(sources, componentmetrics.Source{Feed: "sapient", Role: "projector", Component: a.sapientProjector})
	}
	if a.klvMediaInput != nil {
		sources = append(sources, componentmetrics.Source{Feed: "klv", Role: "media-ref-input", Component: a.klvMediaInput})
	}
	if a.klvDemux != nil {
		sources = append(sources, componentmetrics.Source{Feed: "klv", Role: "demux", Component: a.klvDemux})
	}
	if a.klvDecoder != nil {
		sources = append(sources, componentmetrics.Source{Feed: "klv", Role: "decoder", Component: a.klvDecoder})
	}
	if a.klvProjector != nil {
		sources = append(sources, componentmetrics.Source{Feed: "klv", Role: "projector", Component: a.klvProjector})
	}
	if a.weatherInput != nil {
		sources = append(sources, componentmetrics.Source{Feed: "weather", Role: "fixture-input", Component: a.weatherInput})
	}
	if a.weatherDecoder != nil {
		sources = append(sources, componentmetrics.Source{Feed: "weather", Role: "decoder", Component: a.weatherDecoder})
	}
	if a.weatherProjector != nil {
		sources = append(sources, componentmetrics.Source{Feed: "weather", Role: "projector", Component: a.weatherProjector})
	}
	if a.fusionCandidates != nil {
		sources = append(sources, componentmetrics.Source{Feed: "fusion", Role: "candidate-producer", Component: a.fusionCandidates})
	}
	if a.fusionProjector != nil {
		sources = append(sources, componentmetrics.Source{Feed: "fusion", Role: "projector", Component: a.fusionProjector})
	}
	return sources
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
		if err := app.startSAPIENTFlow(runtimeCtx, cfg.SAPIENT, bindings, deps); err != nil {
			return nil, err
		}
	}

	if cfg.KLV.Enabled {
		if err := app.startKLVFlow(runtimeCtx, cfg.KLV, bindings, deps); err != nil {
			return nil, err
		}
	}

	if cfg.Weather.Enabled {
		if err := app.startWeatherFlow(runtimeCtx, cfg.Weather, bindings, deps); err != nil {
			return nil, err
		}
	}

	if cfg.Fusion.Enabled {
		if err := app.startFusionFlow(runtimeCtx, cfg.Fusion, bindings, deps); err != nil {
			return nil, err
		}
	}
	if cfg.Fusion.CandidateProducerEnabled {
		if err := app.startFusionCandidateFlow(runtimeCtx, cfg.Fusion, deps); err != nil {
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
		newSAPIENTPlanWriter: newSAPIENTPlanWriter,
		newSAPIENTHTTPInput:  sapientcomponent.NewHTTPInputComponent,
		newSAPIENTDecoder:    sapientcomponent.NewDecoderComponent,
		newSAPIENTProjector:  sapientcomponent.NewProjectorComponent,
		newKLVPlanWriter:     newKLVPlanWriter,
		newKLVMediaRefInput:  klvcomponent.NewMediaRefInputComponent,
		newKLVDemux:          klvcomponent.NewDemuxComponent,
		newKLVDecoder:        klvcomponent.NewDecoderComponent,
		newKLVProjector:      klvcomponent.NewProjectorComponent,
		newWeatherPlanWriter: newWeatherPlanWriter,
		newWeatherInput:      weathercomponent.NewFixtureInputComponent,
		newWeatherDecoder:    weathercomponent.NewDecoderComponent,
		newWeatherProjector:  weathercomponent.NewProjectorComponent,
		newFusionCandidates:  fusioncomponent.NewCandidateProducerComponent,
		newFusionPlanWriter:  newFusionPlanWriter,
		newFusionProjector:   fusioncomponent.NewProjectorComponent,
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
	if deps.newSAPIENTPlanWriter == nil {
		deps.newSAPIENTPlanWriter = defaults.newSAPIENTPlanWriter
	}
	if deps.newSAPIENTHTTPInput == nil {
		deps.newSAPIENTHTTPInput = defaults.newSAPIENTHTTPInput
	}
	if deps.newSAPIENTDecoder == nil {
		deps.newSAPIENTDecoder = defaults.newSAPIENTDecoder
	}
	if deps.newSAPIENTProjector == nil {
		deps.newSAPIENTProjector = defaults.newSAPIENTProjector
	}
	if deps.newKLVPlanWriter == nil {
		deps.newKLVPlanWriter = defaults.newKLVPlanWriter
	}
	if deps.newKLVMediaRefInput == nil {
		deps.newKLVMediaRefInput = defaults.newKLVMediaRefInput
	}
	if deps.newKLVDemux == nil {
		deps.newKLVDemux = defaults.newKLVDemux
	}
	if deps.newKLVDecoder == nil {
		deps.newKLVDecoder = defaults.newKLVDecoder
	}
	if deps.newKLVProjector == nil {
		deps.newKLVProjector = defaults.newKLVProjector
	}
	if deps.newWeatherPlanWriter == nil {
		deps.newWeatherPlanWriter = defaults.newWeatherPlanWriter
	}
	if deps.newWeatherInput == nil {
		deps.newWeatherInput = defaults.newWeatherInput
	}
	if deps.newWeatherDecoder == nil {
		deps.newWeatherDecoder = defaults.newWeatherDecoder
	}
	if deps.newWeatherProjector == nil {
		deps.newWeatherProjector = defaults.newWeatherProjector
	}
	if deps.newFusionCandidates == nil {
		deps.newFusionCandidates = defaults.newFusionCandidates
	}
	if deps.newFusionPlanWriter == nil {
		deps.newFusionPlanWriter = defaults.newFusionPlanWriter
	}
	if deps.newFusionProjector == nil {
		deps.newFusionProjector = defaults.newFusionProjector
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

func (a *App) startSAPIENTFlow(
	ctx context.Context,
	cfg SAPIENTConfig,
	bindings copownership.BindingResult,
	deps dependencies,
) error {
	bus := sapientRuntimeBus{client: a.client}
	registry := payloadregistry.New()
	var projector *sapientcomponent.ProjectorComponent
	if cfg.GraphEnabled {
		writer, err := deps.newSAPIENTPlanWriter(cfg.WriteTimeout, cfg.Retry, a.client)
		if err != nil {
			return fmt.Errorf("compose SAPIENT graph writer: %w", err)
		}
		projector, err = deps.newSAPIENTProjector(sapientcomponent.ProjectorConfig{
			Registry: registry,
			Projector: sapientprojector.NewProjector(sapientprojector.Config{
				Org:         cfg.Org,
				Platform:    cfg.Platform,
				OwnerTokens: bindings.OwnerTokenMap(),
				TraceID:     cfg.TraceID,
			}),
			Writer:       writer,
			WriteTimeout: cfg.WriteTimeout,
		}, bus)
		if err != nil {
			return fmt.Errorf("compose SAPIENT projector component: %w", err)
		}
	}
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
	a.sapientProjector = projector
	a.sapientDecoder = decoder
	a.sapientInput = input

	if projector != nil {
		if err := startLifecycle(ctx, "SAPIENT projector", projector); err != nil {
			return err
		}
	}
	if err := startLifecycle(ctx, "SAPIENT decoder", decoder); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "SAPIENT HTTP input", input); err != nil {
		return err
	}
	return nil
}

func (a *App) startKLVFlow(
	ctx context.Context,
	cfg KLVConfig,
	bindings copownership.BindingResult,
	deps dependencies,
) error {
	bus := klvRuntimeBus{client: a.client}
	registry := payloadregistry.New()
	writer, err := deps.newKLVPlanWriter(stack.KLVAdapterConfig{
		Source:       cfg.Source,
		Org:          cfg.Org,
		Platform:     cfg.Platform,
		OwnerTokens:  bindings.OwnerTokenMap(),
		TraceID:      cfg.TraceID,
		WriteTimeout: cfg.WriteTimeout,
		Retry:        cfg.Retry,
	}, stack.KLVAdapterDeps{NATS: a.client})
	if err != nil {
		return fmt.Errorf("compose KLV graph writer: %w", err)
	}

	projector, err := deps.newKLVProjector(klvcomponent.ProjectorConfig{
		Registry: registry,
		Projector: klvprojector.NewProjector(klvprojector.Config{
			Org:         cfg.Org,
			Platform:    cfg.Platform,
			OwnerTokens: bindings.OwnerTokenMap(),
			TraceID:     cfg.TraceID,
		}),
		Writer:       writer,
		WriteTimeout: cfg.WriteTimeout,
		Bus:          bus,
	})
	if err != nil {
		return fmt.Errorf("compose KLV projector component: %w", err)
	}
	decoder, err := deps.newKLVDecoder(klvcomponent.DecoderConfig{
		Source:          klvcomponent.DefaultDecodeSource,
		PacketSubject:   klvcomponent.DefaultPacketSubject,
		FrameSubject:    klvcomponent.DefaultFrameSubject,
		SupportedSubset: klvcomponent.DefaultSupportedSubset,
		MaxPacketBytes:  cfg.Decode.MaxPacketBytes,
		Registry:        registry,
		Bus:             bus,
	})
	if err != nil {
		return fmt.Errorf("compose KLV decoder component: %w", err)
	}
	demux, err := deps.newKLVDemux(klvcomponent.DemuxConfig{
		Source:               klvcomponent.DefaultDemuxSource,
		MediaRefSubject:      klvcomponent.DefaultMediaRefSubject,
		PacketSubject:        klvcomponent.DefaultPacketSubject,
		MaxPacketBytes:       cfg.Demux.MaxPacketBytes,
		MaxExtractBytes:      cfg.Demux.MaxExtractBytes,
		MaxPackets:           cfg.Demux.MaxPackets,
		MaxMaterializedBytes: cfg.Demux.MaxMaterializedBytes,
		ProbeOutputMaxBytes:  cfg.Demux.ProbeOutputMaxBytes,
		Registry:             registry,
		Bus:                  bus,
	})
	if err != nil {
		return fmt.Errorf("compose KLV demux component: %w", err)
	}
	input, err := deps.newKLVMediaRefInput(klvcomponent.MediaRefInputConfig{
		Source:          cfg.Source,
		MediaPath:       cfg.MediaPath,
		MediaPattern:    cfg.MediaPattern,
		MediaRefSubject: klvcomponent.DefaultMediaRefSubject,
		Bus:             bus,
	})
	if err != nil {
		return fmt.Errorf("compose KLV media-ref input component: %w", err)
	}
	a.klvProjector = projector
	a.klvDecoder = decoder
	a.klvDemux = demux
	a.klvMediaInput = input

	if err := startLifecycle(ctx, "KLV projector", projector); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "KLV decoder", decoder); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "KLV demux", demux); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "KLV media-ref input", input); err != nil {
		return err
	}
	return nil
}

func (a *App) startWeatherFlow(
	ctx context.Context,
	cfg WeatherConfig,
	bindings copownership.BindingResult,
	deps dependencies,
) error {
	bus := weatherRuntimeBus{client: a.client}
	registry := payloadregistry.New()
	writer, err := deps.newWeatherPlanWriter(stack.WeatherAdapterConfig{
		Source:       cfg.Source,
		Org:          cfg.Org,
		Platform:     cfg.Platform,
		OwnerTokens:  bindings.OwnerTokenMap(),
		TraceID:      cfg.TraceID,
		WriteTimeout: cfg.WriteTimeout,
		Retry:        cfg.Retry,
	}, stack.WeatherAdapterDeps{NATS: a.client})
	if err != nil {
		return fmt.Errorf("compose weather graph writer: %w", err)
	}

	projector, err := deps.newWeatherProjector(weathercomponent.ProjectorConfig{
		Registry: registry,
		Projector: weatherprojector.NewProjector(weatherprojector.Config{
			Org:         cfg.Org,
			Platform:    cfg.Platform,
			OwnerTokens: bindings.OwnerTokenMap(),
			TraceID:     cfg.TraceID,
		}),
		Writer:          writer,
		WriteTimeout:    cfg.WriteTimeout,
		Freshness:       cfg.Freshness,
		MaxObservations: cfg.MaxObservations,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose weather projector component: %w", err)
	}
	decoder, err := deps.newWeatherDecoder(weathercomponent.DecoderConfig{
		RawSubject:     weathercomponent.DefaultRawSubject,
		DecodedSubject: weathercomponent.DefaultDecodedSubject,
		Registry:       registry,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose weather decoder component: %w", err)
	}
	input, err := deps.newWeatherInput(weathercomponent.FixtureInputConfig{
		Source:      cfg.Source,
		Provider:    cfg.Provider,
		QueryShape:  cfg.QueryShape,
		FixturePath: cfg.FixturePath,
		RawSubject:  weathercomponent.DefaultRawSubject,
		Registry:    registry,
		Bus:         bus,
	})
	if err != nil {
		return fmt.Errorf("compose weather fixture input component: %w", err)
	}
	a.weatherProjector = projector
	a.weatherDecoder = decoder
	a.weatherInput = input

	if err := startLifecycle(ctx, "weather projector", projector); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "weather decoder", decoder); err != nil {
		return err
	}
	if err := startLifecycle(ctx, "weather fixture input", input); err != nil {
		return err
	}
	return nil
}

func (a *App) startFusionFlow(
	ctx context.Context,
	cfg FusionConfig,
	bindings copownership.BindingResult,
	deps dependencies,
) error {
	bus := fusionRuntimeBus{client: a.client}
	registry := payloadregistry.New()
	writer, err := deps.newFusionPlanWriter(cfg.WriteTimeout, cfg.Retry, a.client)
	if err != nil {
		return fmt.Errorf("compose fusion graph writer: %w", err)
	}
	projector, err := deps.newFusionProjector(fusioncomponent.ProjectorConfig{
		CandidateSubject: cfg.CandidateSubject,
		Registry:         registry,
		Association: fusionassociation.Config{
			Org:               cfg.Org,
			Platform:          cfg.Platform,
			MaxDistanceMeters: cfg.AssociationMaxDistanceMeters,
			MaxTimeDelta:      cfg.AssociationMaxTimeDelta,
			MinConfidence:     cfg.AssociationMinConfidence,
			AmbiguityMargin:   cfg.AssociationAmbiguityMargin,
		},
		Projector: fusionprojector.NewProjector(fusionprojector.Config{
			OwnerTokens: bindings.OwnerTokenMap(),
			TraceID:     cfg.TraceID,
		}),
		Writer:       writer,
		WriteTimeout: cfg.WriteTimeout,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose fusion projector component: %w", err)
	}
	a.fusionProjector = projector
	if err := startLifecycle(ctx, "fusion projector", projector); err != nil {
		return err
	}
	return nil
}

func (a *App) startFusionCandidateFlow(
	ctx context.Context,
	cfg FusionConfig,
	deps dependencies,
) error {
	bus := fusionRuntimeBus{client: a.client}
	registry := payloadregistry.New()
	producer, err := deps.newFusionCandidates(fusioncomponent.CandidateProducerConfig{
		CandidateSubject:   cfg.CandidateSubject,
		QuerySubject:       fusioncomponent.SubjectGraphQueryPrefix,
		Registry:           registry,
		Requester:          a.client,
		Sources:            fusionCandidateScopes(cfg),
		PollInterval:       cfg.CandidatePollInterval,
		QueryTimeout:       cfg.CandidateQueryTimeout,
		LimitPerSource:     cfg.CandidateLimitPerSource,
		MaxPairComparisons: cfg.CandidateMaxPairComparisons,
		MaxBatches:         cfg.CandidateMaxBatches,
	}, bus)
	if err != nil {
		return fmt.Errorf("compose fusion candidate producer component: %w", err)
	}
	a.fusionCandidates = producer
	if err := startLifecycle(ctx, "fusion candidate producer", producer); err != nil {
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

func newSAPIENTPlanWriter(
	timeout time.Duration,
	retry natsclient.RetryConfig,
	client graphrequest.RetryRequester,
) (sapientcomponent.PlanWriter, error) {
	if client == nil {
		return nil, fmt.Errorf("sapient stack requires a NATS requester")
	}
	opts := []graphrequest.NATSRequesterOption{}
	if retry != (natsclient.RetryConfig{}) {
		opts = append(opts, graphrequest.WithRetryConfig(retry))
	}
	requester := graphrequest.NewNATSRequester(client, opts...)
	return sapientprojector.NewGraphWriter(
		requester,
		sapientprojector.WithWriteTimeout(timeout),
	), nil
}

func newKLVPlanWriter(
	cfg stack.KLVAdapterConfig,
	deps stack.KLVAdapterDeps,
) (klvcomponent.ProjectorPlanWriter, error) {
	return stack.NewKLVPlanWriter(cfg, deps)
}

func newWeatherPlanWriter(
	cfg stack.WeatherAdapterConfig,
	deps stack.WeatherAdapterDeps,
) (weathercomponent.PlanWriter, error) {
	return stack.NewWeatherPlanWriter(cfg, deps)
}

func newFusionPlanWriter(
	timeout time.Duration,
	retry natsclient.RetryConfig,
	client graphrequest.RetryRequester,
) (fusioncomponent.PlanWriter, error) {
	if client == nil {
		return nil, fmt.Errorf("fusion stack requires a NATS requester")
	}
	opts := []graphrequest.NATSRequesterOption{}
	if retry != (natsclient.RetryConfig{}) {
		opts = append(opts, graphrequest.WithRetryConfig(retry))
	}
	requester := graphrequest.NewNATSRequester(client, opts...)
	return fusionprojector.NewGraphWriter(
		requester,
		fusionprojector.WithWriteTimeout(timeout),
	), nil
}

func fusionCandidateScopes(cfg FusionConfig) []fusioncomponent.CandidateSourceScope {
	sources := cfg.CandidateSources
	if len(sources) == 0 {
		sources = []string{"mavlink", "tak", "adsb", "sapient"}
	}
	scopes := make([]fusioncomponent.CandidateSourceScope, 0, len(sources))
	for _, source := range sources {
		source = strings.ToLower(strings.TrimSpace(source))
		if source == "" {
			continue
		}
		scopes = append(scopes, fusioncomponent.CandidateSourceScope{
			Org:      cfg.Org,
			Platform: cfg.Platform,
			Source:   source,
		})
	}
	return scopes
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
	if cfg.SAPIENT.Enabled && cfg.SAPIENT.GraphEnabled {
		owned = append(owned, cop.OwnedContract{
			Owner:    cop.OwnerSAPIENT,
			Contract: cop.SAPIENTTrackContract(),
		})
	}
	if cfg.KLV.Enabled {
		owned = append(owned, cop.OwnedContract{
			Owner:    cop.OwnerKLV,
			Contract: cop.KLVSensorFootprintContract(),
		})
	}
	if cfg.Weather.Enabled {
		owned = append(owned, cop.OwnedContract{
			Owner:    cop.OwnerWeather,
			Contract: cop.WeatherObservationContract(),
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

type weatherRuntimeBus struct {
	client semstreamsClient
}

func (b weatherRuntimeBus) Publish(ctx context.Context, subject string, data []byte) error {
	return b.client.Publish(ctx, subject, data)
}

func (b weatherRuntimeBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (weathercomponent.Subscription, error) {
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

type fusionRuntimeBus struct {
	client semstreamsClient
}

func (b fusionRuntimeBus) Publish(ctx context.Context, subject string, data []byte) error {
	return b.client.Publish(ctx, subject, data)
}

func (b fusionRuntimeBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (fusioncomponent.Subscription, error) {
	subscription, err := b.client.Subscribe(ctx, subject, handler)
	if err != nil {
		return nil, err
	}
	if subscription == nil {
		return noopSubscription{}, nil
	}
	return subscription, nil
}

type klvRuntimeBus struct {
	client semstreamsClient
}

func (b klvRuntimeBus) Publish(ctx context.Context, subject string, data []byte) error {
	return b.client.Publish(ctx, subject, data)
}

func (b klvRuntimeBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (klvcomponent.Subscription, error) {
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
