//go:build ignore
// +build ignore

package robotics

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/c360/streamkit/natsclient"
	gonats "github.com/nats-io/nats.go"
)

var (
	// sharedTestClient is a single NATS test client shared across all tests
	// This prevents Docker resource exhaustion from creating too many containers
	sharedTestClient *natsclient.TestClient

	// sharedNATSConn is the underlying NATS connection for tests that need it directly
	sharedNATSConn *gonats.Conn
)

// TestMain sets up shared resources for all tests in the package
func TestMain(m *testing.M) {
	// Check for integration test flag
	if os.Getenv("INTEGRATION_TESTS") == "" {
		log.Println("Skipping integration tests. Set INTEGRATION_TESTS=1 to run.")
		return
	}

	// Setup phase
	var exitCode int

	// Create a single shared NATS client for all tests
	// This dramatically reduces Docker container count from 50+ to 1
	log.Println("Setting up shared NATS container for robotics tests...")
	testClient, err := natsclient.NewSharedTestClient(
		natsclient.WithJetStream(),
	)
	if err != nil {
		log.Fatalf("Failed to create shared test client: %v", err)
	}

	// Store globally for tests to use
	sharedTestClient = testClient
	sharedNATSConn = testClient.GetNativeConnection()

	defer func() {
		// Cleanup phase
		// Give time for any async operations to complete
		time.Sleep(100 * time.Millisecond)

		// Terminate the test container
		log.Println("Cleaning up shared NATS container...")
		if err := testClient.Terminate(); err != nil {
			log.Printf("Warning: Failed to terminate test NATS container: %v", err)
		}
	}()

	// Run tests
	exitCode = m.Run()

	os.Exit(exitCode)
}

// getTestNATSConnection returns a NATS connection for testing
// It returns the shared connection when available, or creates a mock otherwise
func getTestNATSConnection(t *testing.T) *gonats.Conn {
	t.Helper()

	if sharedNATSConn != nil {
		// Use the shared connection from TestMain
		return sharedNATSConn
	}

	// Fallback to mock for unit tests or when running with -short
	return mockNATSConnection(t)
}

// getTestClient returns the shared test client for tests that need it
func getTestClient(t *testing.T) *natsclient.TestClient {
	t.Helper()

	if sharedTestClient != nil {
		return sharedTestClient
	}

	// Create a new one if TestMain didn't (e.g., running single test)
	return natsclient.NewTestClient(t, natsclient.WithJetStream())
}
