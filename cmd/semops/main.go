package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/c360studio/semops" // Import for documentation
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

	log.Printf(
		"SemOps runtime started: nats=%s mavlink_enabled=%t cop_owners=%d",
		cfg.NATSURL,
		cfg.MAVLink.Enabled,
		len(runtime.OwnershipBinding().Owners),
	)

	// TODO: Start SOSA API server
	// TODO: Start monitoring services

	log.Println("SemOps initialization complete")

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("SemOps shutdown complete")
}

func closeRuntime(runtime *semopsapp.App, timeout time.Duration) {
	closeCtx, closeCancel := context.WithTimeout(context.Background(), timeout)
	defer closeCancel()
	if err := runtime.Close(closeCtx); err != nil {
		log.Printf("Close SemOps runtime: %v", err)
	}
}
