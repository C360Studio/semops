package stack

import (
	"fmt"
	"time"

	klvcomponent "github.com/c360studio/semops/internal/components/klv"
	"github.com/c360studio/semops/internal/graphrequest"
	klvprojector "github.com/c360studio/semops/internal/projectors/klv"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

type KLVAdapterConfig struct {
	Source string
	Org    string
	// OwnerTokens are minted by SemStreams ownership registration and passed
	// through to the projector without exposing the wire format.
	OwnerTokens  map[string]ownership.OwnerToken
	Platform     string
	TraceID      string
	Confidence   float64
	WriteTimeout time.Duration
	Retry        natsclient.RetryConfig
	Clock        func() time.Time
}

type KLVAdapterDeps struct {
	NATS   graphrequest.RetryRequester
	Writer klvcomponent.ProjectorPlanWriter
}

func NewKLVPlanWriter(
	cfg KLVAdapterConfig,
	deps KLVAdapterDeps,
) (klvcomponent.ProjectorPlanWriter, error) {
	if deps.Writer != nil {
		return deps.Writer, nil
	}
	if deps.NATS == nil {
		return nil, fmt.Errorf("klv stack requires a NATS requester or injected plan writer")
	}

	opts := []graphrequest.NATSRequesterOption{}
	if !isZeroRetryConfig(cfg.Retry) {
		opts = append(opts, graphrequest.WithRetryConfig(cfg.Retry))
	}
	requester := graphrequest.NewNATSRequester(deps.NATS, opts...)
	return klvprojector.NewGraphWriter(
		requester,
		klvprojector.WithWriteTimeout(cfg.WriteTimeout),
	), nil
}
