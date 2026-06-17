//go:build ignore
// +build ignore

package robotics

import (
	"testing"
	"time"

	"github.com/c360/streamkit/metric"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRoboticsProcessorMetrics_Creation(t *testing.T) {
	// Test metrics creation with nil registry (nil input = nil feature pattern)
	metrics := newRoboticsMetrics(nil, "test")
	if metrics != nil {
		t.Fatal("Expected nil metrics when no registry provided (nil input = nil feature)")
	}

	// Test metrics creation with real registry
	registry := metric.NewMetricsRegistry()
	metrics = newRoboticsMetrics(registry, "test")
	if metrics == nil {
		t.Fatal("Expected metrics to be created with registry")
	}

	// Verify metrics are properly initialized
	if metrics.messagesReceived == nil {
		t.Error("messagesReceived metric not initialized")
	}
	if metrics.parseDuration == nil {
		t.Error("parseDuration metric not initialized")
	}
	if metrics.activeVehicles == nil {
		t.Error("activeVehicles metric not initialized")
	}
}

func TestRoboticsProcessorMetrics_NilSafety(t *testing.T) {
	// Create processor without metrics (nil registry)
	testClient := getTestClient(t)

	processor, err := NewRoboticsProcessorWithMetrics(testClient.Client.GetConnection(), nil)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Test that processor handles nil metrics gracefully (nil input = nil feature)
	if processor.metrics != nil {
		t.Error("Expected nil metrics when no registry provided (nil input = nil feature)")
	}

	// Test vehicle tracking with nil metrics (should not panic)
	processor.trackVehicleActivity(1)
	processor.trackVehicleActivity(2)
}

func TestRoboticsProcessorMetrics_Basic(t *testing.T) {
	// Create processor with metrics
	registry := metric.NewMetricsRegistry()
	testClient := getTestClient(t)

	processor, err := NewRoboticsProcessorWithMetrics(testClient.Client.GetConnection(), registry)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Test messages received counter with labels
	if processor.metrics.messagesReceived != nil {
		processor.metrics.messagesReceived.WithLabelValues("raw.drone.mavlink", "mavlink").Inc()
		processor.metrics.messagesReceived.WithLabelValues("raw.drone.json", "json").Inc()

		// Verify the labeled counters
		mavlinkValue := testutil.ToFloat64(processor.metrics.messagesReceived.WithLabelValues("raw.drone.mavlink", "mavlink"))
		jsonValue := testutil.ToFloat64(processor.metrics.messagesReceived.WithLabelValues("raw.drone.json", "json"))

		if mavlinkValue != 1.0 {
			t.Errorf("Expected MAVLink counter to be 1.0, got %f", mavlinkValue)
		}
		if jsonValue != 1.0 {
			t.Errorf("Expected JSON counter to be 1.0, got %f", jsonValue)
		}
	}

	// Test messages processed counter
	if processor.metrics.messagesProcessed != nil {
		processor.metrics.messagesProcessed.WithLabelValues("1", "HEARTBEAT", "mavlink").Inc()
		processor.metrics.messagesProcessed.WithLabelValues("1", "POSITION", "mavlink").Inc()

		heartbeatValue := testutil.ToFloat64(processor.metrics.messagesProcessed.WithLabelValues("1", "HEARTBEAT", "mavlink"))
		positionValue := testutil.ToFloat64(processor.metrics.messagesProcessed.WithLabelValues("1", "POSITION", "mavlink"))

		if heartbeatValue != 1.0 {
			t.Errorf("Expected HEARTBEAT counter to be 1.0, got %f", heartbeatValue)
		}
		if positionValue != 1.0 {
			t.Errorf("Expected POSITION counter to be 1.0, got %f", positionValue)
		}
	}

	// Test parse duration histogram
	if processor.metrics.parseDuration != nil {
		processor.metrics.parseDuration.WithLabelValues("mavlink", "HEARTBEAT").Observe(0.001)
		processor.metrics.parseDuration.WithLabelValues("json", "POSITION").Observe(0.002)

		// Histograms are complex to verify, just ensure they don't panic and can be observed
	}

	// Test active vehicles gauge
	if processor.metrics.activeVehicles != nil {
		processor.metrics.activeVehicles.Set(3)

		value := testutil.ToFloat64(processor.metrics.activeVehicles)
		if value != 3.0 {
			t.Errorf("Expected active vehicles to be 3.0, got %f", value)
		}
	}

	// Test payload size histogram
	if processor.metrics.payloadSize != nil {
		processor.metrics.payloadSize.WithLabelValues("heartbeat").Observe(128)
		processor.metrics.payloadSize.WithLabelValues("position").Observe(256)

		// Histograms store samples, just ensure no panic
	}
}

func TestRoboticsProcessorMetrics_VehicleTracking(t *testing.T) {
	// Create processor with metrics
	registry := metric.NewMetricsRegistry()
	testClient := getTestClient(t)

	processor, err := NewRoboticsProcessorWithMetrics(testClient.Client.GetConnection(), registry)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Test vehicle activity tracking
	processor.trackVehicleActivity(1)
	processor.trackVehicleActivity(2)
	processor.trackVehicleActivity(3)

	// Check that we have 3 active vehicles
	if processor.metrics.activeVehicles != nil {
		value := testutil.ToFloat64(processor.metrics.activeVehicles)
		if value != 3.0 {
			t.Errorf("Expected 3 active vehicles, got %f", value)
		}
	}

	// Check the internal tracking map
	processor.vehiclesMu.RLock()
	if len(processor.knownVehicles) != 3 {
		t.Errorf("Expected 3 vehicles in tracking map, got %d", len(processor.knownVehicles))
	}
	processor.vehiclesMu.RUnlock()

	// Test vehicle activity with same vehicle (should not increase count)
	processor.trackVehicleActivity(1)
	if processor.metrics.activeVehicles != nil {
		value := testutil.ToFloat64(processor.metrics.activeVehicles)
		if value != 3.0 {
			t.Errorf("Expected 3 active vehicles after duplicate, got %f", value)
		}
	}
}

func TestRoboticsProcessorMetrics_ErrorHandling(t *testing.T) {
	// Create processor with metrics
	registry := metric.NewMetricsRegistry()
	testClient := getTestClient(t)

	processor, err := NewRoboticsProcessorWithMetrics(testClient.Client.GetConnection(), registry)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Test dropped messages counter
	if processor.metrics.messagesDropped != nil {
		processor.metrics.messagesDropped.WithLabelValues("1", "parsing_error", "mavlink").Inc()
		processor.metrics.messagesDropped.WithLabelValues("2", "checksum_error", "mavlink").Inc()

		parsingErrorValue := testutil.ToFloat64(processor.metrics.messagesDropped.WithLabelValues("1", "parsing_error", "mavlink"))
		checksumErrorValue := testutil.ToFloat64(processor.metrics.messagesDropped.WithLabelValues("2", "checksum_error", "mavlink"))

		if parsingErrorValue != 1.0 {
			t.Errorf("Expected parsing error counter to be 1.0, got %f", parsingErrorValue)
		}
		if checksumErrorValue != 1.0 {
			t.Errorf("Expected checksum error counter to be 1.0, got %f", checksumErrorValue)
		}
	}

	// Test checksum errors counter
	if processor.metrics.checksumErrors != nil {
		processor.metrics.checksumErrors.WithLabelValues("1", "0").Inc() // Heartbeat message

		value := testutil.ToFloat64(processor.metrics.checksumErrors.WithLabelValues("1", "0"))
		if value != 1.0 {
			t.Errorf("Expected checksum error counter to be 1.0, got %f", value)
		}
	}

	// Test unknown messages counter
	if processor.metrics.unknownMessages != nil {
		processor.metrics.unknownMessages.WithLabelValues("1", "999").Inc() // Unknown message ID

		value := testutil.ToFloat64(processor.metrics.unknownMessages.WithLabelValues("1", "999"))
		if value != 1.0 {
			t.Errorf("Expected unknown message counter to be 1.0, got %f", value)
		}
	}
}

func TestRoboticsProcessorMetrics_TimestampTracking(t *testing.T) {
	// Create processor with metrics
	registry := metric.NewMetricsRegistry()
	testClient := getTestClient(t)

	processor, err := NewRoboticsProcessorWithMetrics(testClient.Client.GetConnection(), registry)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Test last message timestamp tracking
	if processor.metrics.lastMessageTimestamp != nil {
		now := float64(time.Now().Unix())
		processor.metrics.lastMessageTimestamp.WithLabelValues("1", "HEARTBEAT").Set(now)

		value := testutil.ToFloat64(processor.metrics.lastMessageTimestamp.WithLabelValues("1", "HEARTBEAT"))
		if value != now {
			t.Errorf("Expected timestamp to be %f, got %f", now, value)
		}
	}
}
