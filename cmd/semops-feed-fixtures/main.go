package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	sapientcodec "github.com/c360studio/semops/pkg/adapters/sapient"
)

const (
	envFixtureAddr = "SEMOPS_FEED_FIXTURES_ADDR"
	defaultAddr    = ":8091"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		log.Println("Received shutdown signal, terminating feed fixtures...")
		cancel()
	}()

	addr := fixtureAddr(os.Getenv)
	server, err := startFixtureServer(addr)
	if err != nil {
		log.Fatalf("Start SemOps feed fixtures: %v", err)
	}
	defer closeServer(server, 2*time.Second)

	log.Printf("SemOps feed fixtures v%s (commit: %s, built: %s) listening on %s", version, commit, buildDate, addr)
	<-ctx.Done()
}

func fixtureAddr(getenv func(string) string) string {
	if getenv == nil {
		getenv = os.Getenv
	}
	if value := strings.TrimSpace(getenv(envFixtureAddr)); value != "" {
		return value
	}
	return defaultAddr
}

func startFixtureServer(addr string) (*http.Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           fixtureHandler(time.Now),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("SemOps feed fixtures exited: %v", err)
		}
	}()
	return server, nil
}

func fixtureHandler(clock func() time.Time) http.Handler {
	if clock == nil {
		clock = time.Now
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/adsb/states", func(w http.ResponseWriter, _ *http.Request) {
		records, err := adsbcodec.OpenSkyFixtureRecords(clock().UTC())
		if err != nil || len(records) == 0 {
			http.Error(w, "adsb fixture unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(records[0].RawJSON)
	})
	mux.HandleFunc("/sapient/messages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(sapientcodec.TaskAckFixtureJSON())
	})
	return mux
}

func closeServer(server *http.Server, timeout time.Duration) {
	if server == nil {
		return
	}
	closeCtx, closeCancel := context.WithTimeout(context.Background(), timeout)
	defer closeCancel()
	if err := server.Shutdown(closeCtx); err != nil {
		log.Printf("Close SemOps feed fixtures: %v", err)
	}
}
