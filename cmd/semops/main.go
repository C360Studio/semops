package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/c360studio/semops" // Import for documentation
	copapi "github.com/c360studio/semops/internal/api/cop"
	semopsapp "github.com/c360studio/semops/internal/app"
)

// Version information (set by build)
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, terminating...")
		cancel()
	}()

	// Print version information
	fmt.Printf("SemOps v%s (commit: %s, built: %s)\n", version, commit, buildDate)
	fmt.Println("Robotics & Operational Semantics on SemStreams")
	fmt.Println()

	cfg, err := semopsapp.ConfigFromEnv(os.Getenv)
	if err != nil {
		log.Fatalf("Invalid SemOps configuration: %v", err)
	}

	startCtx, startCancel := context.WithTimeout(ctx, cfg.NATSConnectTimeout)
	defer startCancel()
	runtime, err := semopsapp.Start(startCtx, cfg)
	if err != nil {
		log.Fatalf("Start SemOps runtime: %v", err)
	}
	defer closeRuntime(runtime, cfg.ShutdownTimeout)

	apiServer, err := startAPIServer(cfg.APIAddr)
	if err != nil {
		log.Fatalf("Start SemOps API: %v", err)
	}
	defer closeAPIServer(apiServer, cfg.ShutdownTimeout)

	log.Printf(
		"SemOps runtime started: nats=%s api=%s mavlink_enabled=%t cop_owners=%d",
		cfg.NATSURL,
		cfg.APIAddr,
		cfg.MAVLink.Enabled,
		len(runtime.OwnershipBinding().Owners),
	)

	// TODO: Start monitoring services

	log.Println("SemOps initialization complete")

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("SemOps shutdown complete")
}

func startAPIServer(addr string) (*http.Server, error) {
	handler, err := copapi.NewHandler(copapi.NewFixtureProvider(nil))
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("SemOps API listening on %s", listener.Addr())
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("SemOps API exited: %v", err)
		}
	}()
	return server, nil
}

func closeAPIServer(server *http.Server, timeout time.Duration) {
	if server == nil {
		return
	}
	closeCtx, closeCancel := context.WithTimeout(context.Background(), timeout)
	defer closeCancel()
	if err := server.Shutdown(closeCtx); err != nil {
		log.Printf("Close SemOps API: %v", err)
	}
}

func closeRuntime(runtime *semopsapp.App, timeout time.Duration) {
	closeCtx, closeCancel := context.WithTimeout(context.Background(), timeout)
	defer closeCancel()
	if err := runtime.Close(closeCtx); err != nil {
		log.Printf("Close SemOps runtime: %v", err)
	}
}
