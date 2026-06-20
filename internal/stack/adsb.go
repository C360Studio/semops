package stack

import (
	"fmt"
	"time"

	adsbadapter "github.com/c360studio/semops/internal/adapters/adsb"
	"github.com/c360studio/semops/internal/graphrequest"
	adsbprojector "github.com/c360studio/semops/internal/projectors/adsb"
	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

type ADSBAdapterConfig struct {
	Source   string
	Org      string
	Platform string
	// OwnerTokens are minted by SemStreams ownership registration and passed
	// through to the projector without exposing the wire format.
	OwnerTokens   map[string]ownership.OwnerToken
	TraceID       string
	Confidence    float64
	RawMaxRecords int
	RawMaxBytes   int
	WriteTimeout  time.Duration
	Retry         natsclient.RetryConfig
	Replay        adsbadapter.ReplayAppender
	Clock         func() time.Time
}

type ADSBAdapterDeps struct {
	NATS   graphrequest.RetryRequester
	Writer adsbadapter.PlanWriter
}

func NewADSBAdapter(
	cfg ADSBAdapterConfig,
	deps ADSBAdapterDeps,
) (*adsbadapter.Adapter, error) {
	writer, err := adsbWriter(cfg, deps)
	if err != nil {
		return nil, err
	}

	return adsbadapter.NewAdapter(adsbadapter.Config{
		Source: cfg.Source,
		RawLane: adsbcodec.NewRawLane(adsbcodec.RawLaneConfig{
			Source:     cfg.Source,
			MaxRecords: cfg.RawMaxRecords,
			MaxBytes:   cfg.RawMaxBytes,
			Clock:      cfg.Clock,
		}),
		Replay: cfg.Replay,
		Projector: adsbprojector.NewProjector(adsbprojector.Config{
			Org:         cfg.Org,
			Platform:    cfg.Platform,
			OwnerTokens: cfg.OwnerTokens,
			TraceID:     cfg.TraceID,
			Confidence:  cfg.Confidence,
		}),
		Writer: writer,
		Clock:  cfg.Clock,
	})
}

func adsbWriter(
	cfg ADSBAdapterConfig,
	deps ADSBAdapterDeps,
) (adsbadapter.PlanWriter, error) {
	if deps.Writer != nil {
		return deps.Writer, nil
	}
	if deps.NATS == nil {
		return nil, fmt.Errorf("adsb stack requires a NATS requester or injected plan writer")
	}

	opts := []graphrequest.NATSRequesterOption{}
	if !isZeroRetryConfig(cfg.Retry) {
		opts = append(opts, graphrequest.WithRetryConfig(cfg.Retry))
	}
	requester := graphrequest.NewNATSRequester(deps.NATS, opts...)
	return adsbprojector.NewGraphWriter(
		requester,
		adsbprojector.WithWriteTimeout(cfg.WriteTimeout),
	), nil
}
