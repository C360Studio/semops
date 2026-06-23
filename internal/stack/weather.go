package stack

import (
	"fmt"
	"time"

	weathercomponent "github.com/c360studio/semops/internal/components/weather"
	"github.com/c360studio/semops/internal/graphrequest"
	weatherprojector "github.com/c360studio/semops/internal/projectors/weather"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

type WeatherAdapterConfig struct {
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

type WeatherAdapterDeps struct {
	NATS   graphrequest.RetryRequester
	Writer weathercomponent.PlanWriter
}

func NewWeatherPlanWriter(
	cfg WeatherAdapterConfig,
	deps WeatherAdapterDeps,
) (weathercomponent.PlanWriter, error) {
	if deps.Writer != nil {
		return deps.Writer, nil
	}
	if deps.NATS == nil {
		return nil, fmt.Errorf("weather stack requires a NATS requester or injected plan writer")
	}

	opts := []graphrequest.NATSRequesterOption{}
	if !isZeroRetryConfig(cfg.Retry) {
		opts = append(opts, graphrequest.WithRetryConfig(cfg.Retry))
	}
	requester := graphrequest.NewNATSRequester(deps.NATS, opts...)
	return weatherprojector.NewGraphWriter(
		requester,
		weatherprojector.WithWriteTimeout(cfg.WriteTimeout),
	), nil
}
