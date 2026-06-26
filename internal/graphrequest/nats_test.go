package graphrequest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/c360studio/semstreams/natsclient"
)

func TestNATSRequesterUsesRequestWithRetryClassified(t *testing.T) {
	client := &fakeRetryRequester{response: []byte(`{"success":true}`)}
	retry := natsclient.RetryConfig{
		MaxRetries:        7,
		InitialBackoff:    5 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 1.5,
	}
	requester := NewNATSRequester(client, WithRetryConfig(retry))
	ctx := context.WithValue(context.Background(), testContextKey{}, "marker")

	response, err := requester.Request(ctx, "graph.mutation.entity.update_with_triples", []byte("body"), 250*time.Millisecond)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if string(response) != `{"success":true}` {
		t.Fatalf("response = %q", response)
	}
	if client.calls != 1 {
		t.Fatalf("calls = %d, want 1", client.calls)
	}
	if client.ctx != ctx {
		t.Fatal("context was not forwarded")
	}
	if client.subject != "graph.mutation.entity.update_with_triples" {
		t.Fatalf("subject = %q", client.subject)
	}
	if string(client.data) != "body" {
		t.Fatalf("data = %q", client.data)
	}
	if client.timeout != 250*time.Millisecond {
		t.Fatalf("timeout = %s, want 250ms", client.timeout)
	}
	if client.retry != retry {
		t.Fatalf("retry = %+v, want %+v", client.retry, retry)
	}
}

func TestNATSRequesterDefaultsToSemStreamsRetryConfig(t *testing.T) {
	client := &fakeRetryRequester{response: []byte("ok")}
	requester := NewNATSRequester(client)

	if _, err := requester.Request(context.Background(), "subject", nil, 0); err != nil {
		t.Fatalf("request: %v", err)
	}
	if client.retry != natsclient.DefaultRetryConfig() {
		t.Fatalf("retry = %+v, want default %+v", client.retry, natsclient.DefaultRetryConfig())
	}
}

func TestNATSRequesterPropagatesRequestError(t *testing.T) {
	client := &fakeRetryRequester{err: errors.New("no responders")}
	requester := NewNATSRequester(client)

	_, err := requester.Request(context.Background(), "subject", nil, time.Second)
	if err == nil {
		t.Fatal("expected request error")
	}
	if !errors.Is(err, client.err) {
		t.Fatalf("error = %v, want %v", err, client.err)
	}
}

func TestNATSRequesterRequiresClient(t *testing.T) {
	_, err := NewNATSRequester(nil).Request(context.Background(), "subject", nil, time.Second)
	if err == nil {
		t.Fatal("expected missing client error")
	}
}

type testContextKey struct{}

type fakeRetryRequester struct {
	calls    int
	ctx      context.Context
	subject  string
	data     []byte
	timeout  time.Duration
	retry    natsclient.RetryConfig
	response []byte
	err      error
}

func (f *fakeRetryRequester) RequestWithRetryClassified(
	ctx context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
	retry natsclient.RetryConfig,
) ([]byte, error) {
	f.calls++
	f.ctx = ctx
	f.subject = subject
	f.data = append([]byte(nil), data...)
	f.timeout = timeout
	f.retry = retry
	return f.response, f.err
}
