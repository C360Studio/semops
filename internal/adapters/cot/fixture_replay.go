package cot

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
)

const DefaultReplayWriteTimeout = time.Second

type ReplayConfig struct {
	Addr         string
	Events       []cotcodec.Event
	WriteTimeout time.Duration
}

func ReplayUDP(ctx context.Context, cfg ReplayConfig) error {
	conn, err := dialReplay(ctx, "udp", cfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	return writeReplayEvents(conn, cfg)
}

func ReplayTCP(ctx context.Context, cfg ReplayConfig) error {
	conn, err := dialReplay(ctx, "tcp", cfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	return writeReplayEvents(conn, cfg)
}

func dialReplay(ctx context.Context, network string, cfg ReplayConfig) (net.Conn, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.Addr == "" {
		return nil, fmt.Errorf("cot replay %s requires an address", network)
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, network, cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("dial cot replay %s: %w", network, err)
	}
	return conn, nil
}

func writeReplayEvents(conn net.Conn, cfg ReplayConfig) error {
	if len(cfg.Events) == 0 {
		return fmt.Errorf("cot replay requires at least one event")
	}
	timeout := cfg.WriteTimeout
	if timeout == 0 {
		timeout = DefaultReplayWriteTimeout
	}
	if timeout < 0 {
		return fmt.Errorf("cot replay write timeout must be greater than zero")
	}
	for i, event := range cfg.Events {
		raw, err := cotcodec.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal cot replay event %d: %w", i+1, err)
		}
		isTCP := false
		if _, ok := conn.(*net.TCPConn); ok {
			isTCP = true
			raw = bytes.ReplaceAll(raw, []byte("\n"), nil)
		}
		if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			return fmt.Errorf("set cot replay write deadline: %w", err)
		}
		if _, err := conn.Write(raw); err != nil {
			return fmt.Errorf("write cot replay event %d: %w", i+1, err)
		}
		if isTCP {
			if _, err := conn.Write([]byte("\n")); err != nil {
				return fmt.Errorf("write cot replay event %d delimiter: %w", i+1, err)
			}
		}
	}
	return nil
}
