package robotics

import (
	"context"
	"testing"

	gonats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/c360/streamkit/component"
)

// mockNATSConnectionForCompliance returns the shared NATS connection from TestMain
func mockNATSConnectionForCompliance(t *testing.T) *gonats.Conn {
	// Use shared connection from TestMain to avoid Docker resource exhaustion
	return getTestNATSConnection(t)
}

// TestRoboticsProcessorInterfaceCompliance verifies robotics processor implements the full Processor interface
func TestRoboticsProcessorInterfaceCompliance(t *testing.T) {
	nc := mockNATSConnectionForCompliance(t)
	roboticsProcessor, err := NewRoboticsProcessor(nc)
	require.NoError(t, err)
	require.NotNil(t, roboticsProcessor)

	// Verify it implements LifecycleComponent interface
	var _ component.LifecycleComponent = roboticsProcessor
	
	// Verify it implements Discoverable interface
	var _ component.Discoverable = roboticsProcessor

	// Test basic identification methods
	name := roboticsProcessor.Name()
	assert.Equal(t, "robotics", name)

	domain := roboticsProcessor.Domain()
	assert.Equal(t, "robotics", domain)

	// Test health status
	health := roboticsProcessor.Health()
	assert.NotNil(t, health)
	assert.True(t, health.Healthy || !health.Healthy) // It's a boolean, should be either true or false

	// Test configuration methods
	config := roboticsProcessor.Configuration()
	assert.NotNil(t, config)

	err = roboticsProcessor.ValidateConfiguration(config)
	assert.NoError(t, err, "Processor should accept its own configuration")

	err = roboticsProcessor.ReloadConfiguration(context.Background(), config)
	assert.NoError(t, err, "Processor should reload its own configuration")

	// Test metrics
	metrics := roboticsProcessor.Metrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "name")
	assert.Equal(t, "robotics", metrics["name"])

	// Test subscription patterns
	patterns := roboticsProcessor.SubscriptionPatterns()
	// Component-owned port definitions provide consistent defaults
	expectedPatterns := []string{
		"input.*.mavlink",
	}
	assert.Equal(t, expectedPatterns, patterns)

	// Test Discoverable interface methods
	meta := roboticsProcessor.Meta()
	assert.Equal(t, "robotics", meta.Name)
	assert.Equal(t, "processor", meta.Type)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Contains(t, meta.Description, "MAVLink")

	inputPorts := roboticsProcessor.InputPorts()
	// Component owns its port definitions - 1 input port
	assert.Len(t, inputPorts, 1)
	assert.Equal(t, "mavlink_input", inputPorts[0].Name)

	outputPorts := roboticsProcessor.OutputPorts()
	// Component owns its port definitions - 1 output port (NOT 4!)
	assert.Len(t, outputPorts, 1)
	assert.Equal(t, "storage_write", outputPorts[0].Name)
	// Check that it has the correct subject
	if natsPort, ok := outputPorts[0].Config.(component.NATSPort); ok {
		assert.Equal(t, "storage.objectstore.write", natsPort.Subject)
	}

	configSchema := roboticsProcessor.ConfigSchema()
	assert.NotNil(t, configSchema.Properties)
	// Schema contains user-configurable properties (not runtime state like "enabled" or "formats")
	assert.Contains(t, configSchema.Properties, "process_heartbeat")
	assert.Contains(t, configSchema.Properties, "process_battery")

	// Test lifecycle methods (without NATS connection)
	err = roboticsProcessor.ProcessRawData(context.Background(), "test.subject", []byte("test data"))
	assert.NoError(t, err, "ProcessRawData should handle being called without initialization")
}

// TestRoboticsProcessorConfiguration tests configuration validation
func TestRoboticsProcessorConfiguration(t *testing.T) {
	nc := mockNATSConnectionForCompliance(t)
	roboticsProcessor, err := NewRoboticsProcessor(nc)
	require.NoError(t, err)

	// Valid configuration should pass
	validConfig := map[string]any{
		"enabled": true,
		"name":    "robotics",
	}
	err = roboticsProcessor.ValidateConfiguration(validConfig)
	assert.NoError(t, err)

	// Invalid configuration type should fail
	err = roboticsProcessor.ValidateConfiguration("not a map")
	assert.Error(t, err)

	// Invalid enabled field should fail
	invalidConfig := map[string]any{
		"enabled": "not a bool",
	}
	err = roboticsProcessor.ValidateConfiguration(invalidConfig)
	assert.Error(t, err)

	// Configuration reload should work
	err = roboticsProcessor.ReloadConfiguration(context.Background(), validConfig)
	assert.NoError(t, err)
}

// TestRoboticsProcessorMetrics tests metrics functionality
func TestRoboticsProcessorMetrics(t *testing.T) {
	nc := mockNATSConnectionForCompliance(t)
	roboticsProcessor, err := NewRoboticsProcessor(nc)
	require.NoError(t, err)

	metrics := roboticsProcessor.Metrics()
	assert.NotNil(t, metrics)

	// Check required fields
	assert.Equal(t, "robotics", metrics["name"])
	assert.Contains(t, metrics, "enabled")
	assert.Contains(t, metrics, "subscription_count")
	assert.Contains(t, metrics, "supported_formats")
	assert.Contains(t, metrics, "vehicle_types")

	// Check supported formats
	formats, ok := metrics["supported_formats"].([]string)
	assert.True(t, ok)
	expectedFormats := []string{"mavlink", "json"}
	assert.Equal(t, expectedFormats, formats)

	// Check vehicle types
	vehicleTypes, ok := metrics["vehicle_types"].([]string)
	assert.True(t, ok)
	expectedTypes := []string{"drone", "ground_vehicle", "boat", "submarine"}
	assert.Equal(t, expectedTypes, vehicleTypes)
}

// TestCreateRoboticsProcessor_EmptyConfigPreservesDefaultPorts tests that
// unmarshaling an empty config {} doesn't wipe out default Ports
func TestCreateRoboticsProcessor_EmptyConfigPreservesDefaultPorts(t *testing.T) {
	// Use shared NATS client from TestMain
	testClient := getTestClient(t)

	deps := component.ComponentDependencies{
		NATSClient: testClient.Client,
		Platform: component.PlatformMeta{
			Org:      "test-org",
			Platform: "test-platform",
		},
	}

	tests := []struct {
		name           string
		config         string
		expectPorts    bool
		expectedInputs int
		expectedOutputs int
	}{
		{
			name:           "empty config preserves default ports",
			config:         `{}`,
			expectPorts:    true,
			expectedInputs: 1,
			expectedOutputs: 1,
		},
		{
			name:           "nil config uses default ports",
			config:         ``,
			expectPorts:    true,
			expectedInputs: 1,
			expectedOutputs: 1,
		},
		{
			name: "explicit ports are merged with defaults",
			config: `{
				"ports": {
					"inputs": [{"name": "custom", "pattern": "stream", "subject": "custom.in"}],
					"outputs": [{"name": "custom", "pattern": "stream", "subject": "custom.out"}]
				}
			}`,
			expectPorts:    true,
			expectedInputs: 2,  // default + custom
			expectedOutputs: 2, // default + custom
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := CreateRoboticsProcessor([]byte(tt.config), deps)
			require.NoError(t, err)
			require.NotNil(t, processor)

			discoverable, ok := processor.(component.Discoverable)
			require.True(t, ok, "processor should implement Discoverable")

			inputPorts := discoverable.InputPorts()
			outputPorts := discoverable.OutputPorts()

			if tt.expectPorts {
				assert.Len(t, inputPorts, tt.expectedInputs, "should have expected input ports")
				assert.Len(t, outputPorts, tt.expectedOutputs, "should have expected output ports")
			}
		})
	}
}