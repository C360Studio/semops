package sapient

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	sapientcomponent "github.com/c360studio/semops/internal/components/sapient"
	sapientcodec "github.com/c360studio/semops/pkg/adapters/sapient"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

const (
	livePreflightNATSEnv            = "SEMOPS_SAPIENT_LIVE_PREFLIGHT_NATS_URL"
	livePreflightExpectedContentEnv = "SEMOPS_SAPIENT_LIVE_PREFLIGHT_EXPECTED_CONTENT"
)

func TestLiveSAPIENTPreflightDecodedSmoke(t *testing.T) {
	natsURL := os.Getenv(livePreflightNATSEnv)
	if natsURL == "" {
		t.Skipf("set %s to run the live SAPIENT preflight smoke", livePreflightNATSEnv)
	}
	expectedContent, err := expectedContentFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := natsclient.NewClient(
		natsURL,
		natsclient.WithName("semops-sapient-live-preflight-smoke"),
		natsclient.WithTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer closeCancel()
		if err := client.Close(closeCtx); err != nil {
			t.Logf("close nats client: %v", err)
		}
	})

	registry := payloadregistry.New()
	if err := sapientcomponent.RegisterPayloads(registry); err != nil {
		t.Fatalf("register SAPIENT payloads: %v", err)
	}
	decoder := message.NewDecoder(registry)
	decoded := make(chan *sapientcomponent.DecodedMessagePayload, 1)
	decodeErr := make(chan error, 1)

	sub, err := client.Subscribe(ctx, sapientcomponent.DefaultDecodedSubject, func(_ context.Context, msg *nats.Msg) {
		envelope, err := decoder.Decode(msg.Data)
		if err != nil {
			trySendErr(decodeErr, fmt.Errorf("decode SAPIENT BaseMessage: %w", err))
			return
		}
		payload, ok := envelope.Payload().(*sapientcomponent.DecodedMessagePayload)
		if !ok {
			trySendErr(decodeErr, fmt.Errorf("decoded payload = %T, want *sapient.DecodedMessagePayload", envelope.Payload()))
			return
		}
		if payload.Content != expectedContent ||
			payload.NodeID != "a8654cdf-4328-47de-81fa-c495589e30c8" ||
			payload.RawRef == "" {
			trySendErr(decodeErr, fmt.Errorf("unexpected SAPIENT decoded payload: %+v", payload))
			return
		}
		select {
		case decoded <- payload:
		default:
		}
	})
	if err != nil {
		t.Fatalf("subscribe SAPIENT decoded subject: %v", err)
	}
	t.Cleanup(func() {
		if err := sub.Unsubscribe(); err != nil {
			t.Logf("unsubscribe SAPIENT decoded subject: %v", err)
		}
	})

	select {
	case <-decoded:
	case err := <-decodeErr:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatalf("SAPIENT preflight decoded payload not observed before timeout: %v", ctx.Err())
	}
}

func expectedContentFromEnv() (sapientcodec.ContentKind, error) {
	value := sapientcodec.ContentKind(os.Getenv(livePreflightExpectedContentEnv))
	if value == "" {
		return sapientcodec.ContentTaskAck, nil
	}
	switch value {
	case sapientcodec.ContentTaskAck, sapientcodec.ContentDetectionReport:
		return value, nil
	default:
		return "", fmt.Errorf(
			"%s=%q is unsupported; expected %q or %q",
			livePreflightExpectedContentEnv,
			value,
			sapientcodec.ContentTaskAck,
			sapientcodec.ContentDetectionReport,
		)
	}
}

func trySendErr(ch chan<- error, err error) {
	select {
	case ch <- err:
	default:
	}
}
