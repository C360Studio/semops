package graphrequest

import (
	"context"
	"fmt"
	"time"

	"github.com/c360studio/semstreams/natsclient"
)

type RetryRequester interface {
	RequestWithRetryClassified(
		ctx context.Context,
		subject string,
		data []byte,
		timeout time.Duration,
		retry natsclient.RetryConfig,
	) ([]byte, error)
}

type NATSRequester struct {
	client RetryRequester
	retry  natsclient.RetryConfig
}

type NATSRequesterOption func(*NATSRequester)

func NewNATSRequester(client RetryRequester, opts ...NATSRequesterOption) *NATSRequester {
	requester := &NATSRequester{
		client: client,
		retry:  natsclient.DefaultRetryConfig(),
	}
	for _, opt := range opts {
		opt(requester)
	}
	return requester
}

func WithRetryConfig(retry natsclient.RetryConfig) NATSRequesterOption {
	return func(requester *NATSRequester) {
		requester.retry = retry
	}
}

func (r *NATSRequester) Request(
	ctx context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
) ([]byte, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("semstreams NATS requester has no client")
	}
	return r.client.RequestWithRetryClassified(ctx, subject, data, timeout, r.retry)
}
