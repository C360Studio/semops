package mavlink

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
)

func TestUDPListenerFeedsDatagramsIntoAdapter(t *testing.T) {
	writer := &notifyingPlanWriter{applied: make(chan mavprojector.Plan, 1)}
	adapter := newTestAdapter(t, writer, time.Now)

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

	conn, err := net.Dial("udp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial listener: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte{0x00, 0x01, 0x02}); err != nil {
		t.Fatalf("write invalid datagram: %v", err)
	}
	waitFor(t, func() bool {
		return adapter.Health().ParseErrors == 1
	})

	generator := mavcodec.NewGenerator(42, 7)
	frame, err := generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{
		BaseMode:       mavcodec.ModeFlagSafetyArmed,
		SystemStatus:   mavcodec.StateActive,
		MavlinkVersion: mavcodec.Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	if _, err := conn.Write(frame); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	select {
	case plan := <-writer.applied:
		if len(plan.Mutations) != 2 {
			t.Fatalf("mutations = %d, want asset + track birth", len(plan.Mutations))
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for graph write")
	}

	health := adapter.Health()
	if !health.Ready {
		t.Fatalf("health ready = false, last error %q", health.LastError)
	}
	if health.FramesReceived != 2 || health.ParseErrors != 1 || health.GraphMutations != 2 {
		t.Fatalf(
			"health frames/errors/mutations = %d/%d/%d, want 2/1/2",
			health.FramesReceived,
			health.ParseErrors,
			health.GraphMutations,
		)
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

func TestListenUDPRejectsInvalidConfig(t *testing.T) {
	adapter := newTestAdapter(t, &recordingPlanWriter{}, time.Now)
	tests := []struct {
		name    string
		cfg     UDPListenerConfig
		adapter *Adapter
	}{
		{name: "nil adapter", cfg: UDPListenerConfig{ListenAddr: "127.0.0.1:0"}},
		{name: "empty address", adapter: adapter},
		{
			name: "negative max datagram",
			cfg: UDPListenerConfig{
				ListenAddr:       "127.0.0.1:0",
				MaxDatagramBytes: -1,
			},
			adapter: adapter,
		},
		{
			name: "negative read interval",
			cfg: UDPListenerConfig{
				ListenAddr:   "127.0.0.1:0",
				ReadInterval: -1,
			},
			adapter: adapter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := ListenUDP(tt.cfg, tt.adapter)
			if listener != nil {
				t.Cleanup(func() {
					if err := listener.Close(); err != nil {
						t.Fatalf("close listener: %v", err)
					}
				})
			}
			if err == nil {
				t.Fatal("expected config error")
			}
		})
	}
}

type notifyingPlanWriter struct {
	mu      sync.Mutex
	plans   []mavprojector.Plan
	applied chan mavprojector.Plan
}

func (w *notifyingPlanWriter) Apply(_ context.Context, plan mavprojector.Plan) error {
	w.mu.Lock()
	w.plans = append(w.plans, plan)
	w.mu.Unlock()

	select {
	case w.applied <- plan:
	default:
	}
	return nil
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for condition")
		case <-ticker.C:
		}
	}
}
