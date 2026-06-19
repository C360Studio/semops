package cot

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
)

const DefaultTCPMaxEventBytes = 1024 * 1024

type TCPListenerConfig struct {
	ListenAddr    string
	MaxEventBytes int
}

type TCPListener struct {
	listener      net.Listener
	adapter       *Adapter
	maxEventBytes int
}

func ListenTCP(cfg TCPListenerConfig, adapter *Adapter) (*TCPListener, error) {
	if adapter == nil {
		return nil, fmt.Errorf("cot tcp listener requires an adapter")
	}
	if cfg.ListenAddr == "" {
		return nil, fmt.Errorf("cot tcp listener requires a listen address")
	}
	if cfg.MaxEventBytes == 0 {
		cfg.MaxEventBytes = DefaultTCPMaxEventBytes
	}
	if cfg.MaxEventBytes < 0 {
		return nil, fmt.Errorf("cot tcp max event bytes must be greater than zero")
	}
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("listen cot tcp: %w", err)
	}
	return &TCPListener{listener: listener, adapter: adapter, maxEventBytes: cfg.MaxEventBytes}, nil
}

func (l *TCPListener) Addr() net.Addr {
	if l == nil || l.listener == nil {
		return nil
	}
	return l.listener.Addr()
}

func (l *TCPListener) Run(ctx context.Context) error {
	if l == nil || l.listener == nil {
		return fmt.Errorf("cot tcp listener is not open")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	for {
		conn, err := l.listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept cot tcp connection: %w", err)
		}
		go l.handleConn(ctx, conn)
	}
}

func (l *TCPListener) Close() error {
	if l == nil || l.listener == nil {
		return nil
	}
	if err := l.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}

func (l *TCPListener) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024), l.maxEventBytes)
	for scanner.Scan() {
		raw := append([]byte(nil), scanner.Bytes()...)
		_, _ = l.adapter.IngestEvent(ctx, raw)
	}
}
