package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/c360studio/semops" // Import for documentation
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

	// TODO: Load configuration
	// TODO: Initialize semstreams clients (EntityStore, GraphProcessor, etc.)
	// TODO: Start protocol adapters (MAVLink, TAK, NMEA)
	// TODO: Start SOSA API server
	// TODO: Start monitoring services

	log.Println("SemOps initialization complete")

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("SemOps shutdown complete")
}
