package stack

import (
	"fmt"
	"time"

	capcomponent "github.com/c360studio/semops/internal/components/cap"
	"github.com/c360studio/semops/internal/graphrequest"
	capprojector "github.com/c360studio/semops/internal/projectors/cap"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

type CAPAdapterConfig struct {
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

type CAPAdapterDeps struct {
	NATS   graphrequest.RetryRequester
	Writer capcomponent.PlanWriter
}

func NewCAPPlanWriter(
	cfg CAPAdapterConfig,
	deps CAPAdapterDeps,
) (capcomponent.PlanWriter, error) {
	return capWriter(cfg, deps)
}

func capWriter(
	cfg CAPAdapterConfig,
	deps CAPAdapterDeps,
) (capcomponent.PlanWriter, error) {
	if deps.Writer != nil {
		return deps.Writer, nil
	}
	if deps.NATS == nil {
		return nil, fmt.Errorf("cap stack requires a NATS requester or injected plan writer")
	}

	opts := []graphrequest.NATSRequesterOption{}
	if !isZeroRetryConfig(cfg.Retry) {
		opts = append(opts, graphrequest.WithRetryConfig(cfg.Retry))
	}
	requester := graphrequest.NewNATSRequester(deps.NATS, opts...)
	return capprojector.NewGraphWriter(
		requester,
		capprojector.WithWriteTimeout(cfg.WriteTimeout),
	), nil
}
