package stack

import (
	"fmt"
	"time"

	cotadapter "github.com/c360studio/semops/internal/adapters/cot"
	"github.com/c360studio/semops/internal/graphrequest"
	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

type CoTAdapterConfig struct {
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
	Replay        cotadapter.ReplayAppender
	Clock         func() time.Time
}

type CoTAdapterDeps struct {
	NATS   graphrequest.RetryRequester
	Writer cotadapter.PlanWriter
}

func NewCoTAdapter(
	cfg CoTAdapterConfig,
	deps CoTAdapterDeps,
) (*cotadapter.Adapter, error) {
	writer, err := cotWriter(cfg, deps)
	if err != nil {
		return nil, err
	}

	return cotadapter.NewAdapter(cotadapter.Config{
		Source: cfg.Source,
		RawLane: cotcodec.NewRawLane(cotcodec.RawLaneConfig{
			Source:     cfg.Source,
			MaxRecords: cfg.RawMaxRecords,
			MaxBytes:   cfg.RawMaxBytes,
			Clock:      cfg.Clock,
		}),
		Replay: cfg.Replay,
		Projector: cotprojector.NewProjector(cotprojector.Config{
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

func cotWriter(
	cfg CoTAdapterConfig,
	deps CoTAdapterDeps,
) (cotadapter.PlanWriter, error) {
	if deps.Writer != nil {
		return deps.Writer, nil
	}
	if deps.NATS == nil {
		return nil, fmt.Errorf("cot stack requires a NATS requester or injected plan writer")
	}

	opts := []graphrequest.NATSRequesterOption{}
	if !isZeroRetryConfig(cfg.Retry) {
		opts = append(opts, graphrequest.WithRetryConfig(cfg.Retry))
	}
	requester := graphrequest.NewNATSRequester(deps.NATS, opts...)
	return cotprojector.NewGraphWriter(
		requester,
		cotprojector.WithWriteTimeout(cfg.WriteTimeout),
	), nil
}
