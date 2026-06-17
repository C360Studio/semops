package stack

import (
	"fmt"
	"time"

	mavadapter "github.com/c360studio/semops/internal/adapters/mavlink"
	"github.com/c360studio/semops/internal/graphrequest"
	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semstreams/natsclient"
)

type MAVLinkAdapterConfig struct {
	Source           string
	Org              string
	Platform         string
	OwnerTokenSuffix string
	TraceID          string
	Confidence       float64
	RawMaxRecords    int
	RawMaxBytes      int
	WriteTimeout     time.Duration
	Retry            natsclient.RetryConfig
	Clock            func() time.Time
}

type MAVLinkAdapterDeps struct {
	NATS   graphrequest.RetryRequester
	Writer mavadapter.PlanWriter
}

func NewMAVLinkAdapter(
	cfg MAVLinkAdapterConfig,
	deps MAVLinkAdapterDeps,
) (*mavadapter.Adapter, error) {
	writer, err := mavlinkWriter(cfg, deps)
	if err != nil {
		return nil, err
	}

	return mavadapter.NewAdapter(mavadapter.Config{
		Source: cfg.Source,
		Parser: mavcodec.NewParser(),
		RawLane: mavcodec.NewRawLane(mavcodec.RawLaneConfig{
			Source:     cfg.Source,
			MaxRecords: cfg.RawMaxRecords,
			MaxBytes:   cfg.RawMaxBytes,
			Clock:      cfg.Clock,
		}),
		Projector: mavprojector.NewProjector(mavprojector.Config{
			Org:              cfg.Org,
			Platform:         cfg.Platform,
			OwnerTokenSuffix: cfg.OwnerTokenSuffix,
			TraceID:          cfg.TraceID,
			Confidence:       cfg.Confidence,
		}),
		Writer: writer,
		Clock:  cfg.Clock,
	})
}

func mavlinkWriter(
	cfg MAVLinkAdapterConfig,
	deps MAVLinkAdapterDeps,
) (mavadapter.PlanWriter, error) {
	if deps.Writer != nil {
		return deps.Writer, nil
	}
	if deps.NATS == nil {
		return nil, fmt.Errorf("mavlink stack requires a NATS requester or injected plan writer")
	}

	opts := []graphrequest.NATSRequesterOption{}
	if !isZeroRetryConfig(cfg.Retry) {
		opts = append(opts, graphrequest.WithRetryConfig(cfg.Retry))
	}
	requester := graphrequest.NewNATSRequester(deps.NATS, opts...)
	return mavprojector.NewGraphWriter(
		requester,
		mavprojector.WithWriteTimeout(cfg.WriteTimeout),
	), nil
}

func isZeroRetryConfig(retry natsclient.RetryConfig) bool {
	return retry.MaxRetries == 0 &&
		retry.InitialBackoff == 0 &&
		retry.MaxBackoff == 0 &&
		retry.BackoffMultiplier == 0
}
