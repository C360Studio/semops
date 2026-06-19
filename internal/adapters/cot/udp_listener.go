package cot

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	DefaultUDPMaxDatagramBytes = 64 * 1024
	DefaultUDPReadInterval     = 250 * time.Millisecond
)

type UDPListenerConfig struct {
	ListenAddr       string
	MaxDatagramBytes int
	ReadInterval     time.Duration
}

type UDPListener struct {
	conn             *net.UDPConn
	adapter          *Adapter
	maxDatagramBytes int
	readInterval     time.Duration
}

func ListenUDP(cfg UDPListenerConfig, adapter *Adapter) (*UDPListener, error) {
	if adapter == nil {
		return nil, fmt.Errorf("cot udp listener requires an adapter")
	}
	if cfg.ListenAddr == "" {
		return nil, fmt.Errorf("cot udp listener requires a listen address")
	}
	if cfg.MaxDatagramBytes == 0 {
		cfg.MaxDatagramBytes = DefaultUDPMaxDatagramBytes
	}
	if cfg.MaxDatagramBytes < 0 {
		return nil, fmt.Errorf("cot udp max datagram bytes must be greater than zero")
	}
	if cfg.ReadInterval == 0 {
		cfg.ReadInterval = DefaultUDPReadInterval
	}
	if cfg.ReadInterval < 0 {
		return nil, fmt.Errorf("cot udp read interval must be greater than zero")
	}

	addr, err := net.ResolveUDPAddr("udp", cfg.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve cot udp address: %w", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen cot udp: %w", err)
	}
	return &UDPListener{
		conn:             conn,
		adapter:          adapter,
		maxDatagramBytes: cfg.MaxDatagramBytes,
		readInterval:     cfg.ReadInterval,
	}, nil
}

func (l *UDPListener) Addr() net.Addr {
	if l == nil || l.conn == nil {
		return nil
	}
	return l.conn.LocalAddr()
}

func (l *UDPListener) Run(ctx context.Context) error {
	if l == nil || l.conn == nil {
		return fmt.Errorf("cot udp listener is not open")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	buffer := make([]byte, l.maxDatagramBytes)
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if err := l.conn.SetReadDeadline(time.Now().Add(l.readInterval)); err != nil {
			return fmt.Errorf("set cot udp read deadline: %w", err)
		}
		n, _, err := l.conn.ReadFromUDP(buffer)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			return fmt.Errorf("read cot udp datagram: %w", err)
		}
		raw := append([]byte(nil), buffer[:n]...)
		_, _ = l.adapter.IngestEvent(ctx, raw)
	}
}

func (l *UDPListener) Close() error {
	if l == nil || l.conn == nil {
		return nil
	}
	if err := l.conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}
