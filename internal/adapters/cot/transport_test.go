package cot

import (
	"context"
	"net"
	"testing"
	"time"

	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
)

func TestUDPFixtureReplayFeedsEventsIntoAdapter(t *testing.T) {
	now := time.Date(2026, 6, 19, 14, 25, 0, 0, time.UTC)
	adapter, seen := newNotifyingAdapter(t, now)
	listener, err := ListenUDP(UDPListenerConfig{
		ListenAddr:   "127.0.0.1:0",
		ReadInterval: 10 * time.Millisecond,
	}, adapter)
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- listener.Run(ctx)
	}()

	events := cotcodec.SeedEvents(now)
	if err := ReplayUDP(ctx, ReplayConfig{Addr: listener.Addr().String(), Events: events}); err != nil {
		t.Fatalf("replay udp: %v", err)
	}
	assertSeenUIDs(t, seen, []string{"ANDROID-ALPHA", "ANDROID-BRAVO", "MARKER-NORTH-GATE", "CHAT-ALPHA-1"})

	health := adapter.Health()
	if !health.Ready || health.EventsReceived != 4 || health.EventsDecoded != 4 || health.ParseErrors != 0 {
		t.Fatalf("health = %+v", health)
	}
	if got := len(adapter.RawLane().Snapshot()); got != 4 {
		t.Fatalf("raw lane records = %d, want 4", got)
	}

	cancel()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for listener shutdown")
	}
}

func TestTCPFixtureReplayFeedsEventsIntoAdapter(t *testing.T) {
	now := time.Date(2026, 6, 19, 14, 26, 0, 0, time.UTC)
	adapter, seen := newNotifyingAdapter(t, now)
	listener, err := ListenTCP(TCPListenerConfig{ListenAddr: "127.0.0.1:0"}, adapter)
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- listener.Run(ctx)
	}()

	events := cotcodec.SeedEvents(now)
	if err := ReplayTCP(ctx, ReplayConfig{Addr: listener.Addr().String(), Events: events}); err != nil {
		t.Fatalf("replay tcp: %v", err)
	}
	assertSeenUIDs(t, seen, []string{"ANDROID-ALPHA", "ANDROID-BRAVO", "MARKER-NORTH-GATE", "CHAT-ALPHA-1"})

	health := adapter.Health()
	if !health.Ready || health.EventsReceived != 4 || health.EventsDecoded != 4 || health.ParseErrors != 0 {
		t.Fatalf("health = %+v", health)
	}

	cancel()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for listener shutdown")
	}
}

func TestListenersRejectInvalidConfig(t *testing.T) {
	adapter, err := NewAdapter(Config{Source: "tak:test"})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "udp nil adapter", run: func() error { _, err := ListenUDP(UDPListenerConfig{ListenAddr: "127.0.0.1:0"}, nil); return err }},
		{name: "udp empty address", run: func() error { _, err := ListenUDP(UDPListenerConfig{}, adapter); return err }},
		{name: "udp negative max", run: func() error {
			_, err := ListenUDP(UDPListenerConfig{ListenAddr: "127.0.0.1:0", MaxDatagramBytes: -1}, adapter)
			return err
		}},
		{name: "tcp nil adapter", run: func() error { _, err := ListenTCP(TCPListenerConfig{ListenAddr: "127.0.0.1:0"}, nil); return err }},
		{name: "tcp empty address", run: func() error { _, err := ListenTCP(TCPListenerConfig{}, adapter); return err }},
		{name: "tcp negative max", run: func() error {
			_, err := ListenTCP(TCPListenerConfig{ListenAddr: "127.0.0.1:0", MaxEventBytes: -1}, adapter)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil {
				t.Fatal("expected config error")
			}
		})
	}
}

func TestReplayRejectsInvalidConfig(t *testing.T) {
	ctx := context.Background()
	if err := ReplayUDP(ctx, ReplayConfig{}); err == nil {
		t.Fatal("expected UDP replay address error")
	}
	if err := ReplayTCP(ctx, ReplayConfig{}); err == nil {
		t.Fatal("expected TCP replay address error")
	}
	if err := writeReplayEvents(discardConn{}, ReplayConfig{}); err == nil {
		t.Fatal("expected empty replay event error")
	}
	if err := writeReplayEvents(discardConn{}, ReplayConfig{Events: cotcodec.SeedEvents(time.Now()), WriteTimeout: -1}); err == nil {
		t.Fatal("expected negative replay timeout error")
	}
}

func newNotifyingAdapter(t *testing.T, now time.Time) (*Adapter, chan IngestResult) {
	t.Helper()
	seen := make(chan IngestResult, 8)
	adapter, err := NewAdapter(Config{
		Source: "tak:fixture",
		Clock:  func() time.Time { return now },
		OnEvent: func(result IngestResult) {
			seen <- result
		},
	})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	return adapter, seen
}

func assertSeenUIDs(t *testing.T, seen <-chan IngestResult, want []string) {
	t.Helper()
	got := make([]string, 0, len(want))
	deadline := time.After(time.Second)
	for len(got) < len(want) {
		select {
		case result := <-seen:
			got = append(got, result.Event.UID)
		case <-deadline:
			t.Fatalf("timed out waiting for events; got %v, want %v", got, want)
		}
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("uid[%d] = %q, want %q (all got %v)", i, got[i], want[i], got)
		}
	}
}

type discardConn struct{}

func (discardConn) Read([]byte) (int, error)         { return 0, nil }
func (discardConn) Write(p []byte) (int, error)      { return len(p), nil }
func (discardConn) Close() error                     { return nil }
func (discardConn) LocalAddr() net.Addr              { return netAddr("local") }
func (discardConn) RemoteAddr() net.Addr             { return netAddr("remote") }
func (discardConn) SetDeadline(time.Time) error      { return nil }
func (discardConn) SetReadDeadline(time.Time) error  { return nil }
func (discardConn) SetWriteDeadline(time.Time) error { return nil }

type netAddr string

func (n netAddr) Network() string { return string(n) }
func (n netAddr) String() string  { return string(n) }
