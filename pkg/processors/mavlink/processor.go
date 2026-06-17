//go:build ignore
// +build ignore

// Package robotics provides autonomous vehicle data processing for the SemStreams platform.
// It supports MAVLink protocol parsing for drones, ground vehicles, boats, and submarines.
package robotics

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	processor "github.com/c360/semstreams/processor/base"
	"github.com/c360/streamkit/component"
	"github.com/c360/streamkit/errors"
	"github.com/c360/streamkit/metric"
	gonats "github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/c360/semops/pkg/processors/mavlink/parser"
	"github.com/c360/semops/pkg/processors/mavlink/payloads"
	messages "github.com/c360/semstreams/message"
)

// Config defines configuration for the RoboticsProcessor
type Config struct {
	// MAVLink filtering
	SystemIDFilter    []uint8 `json:"system_id_filter"`    // Empty = all systems
	ComponentIDFilter []uint8 `json:"component_id_filter"` // Empty = all components

	// Message type enables
	ProcessHeartbeat bool `json:"process_heartbeat"`
	ProcessPosition  bool `json:"process_position"`
	ProcessBattery   bool `json:"process_battery"`
	ProcessStatus    bool `json:"process_status"`
	ProcessAttitude  bool `json:"process_attitude"`

	// Dynamic port configuration (optional - overrides conventions)
	Ports      *component.PortConfig `json:"ports,omitempty"`
	ProcessGPS bool                  `json:"process_gps"`

	// Processing options
	ValidateChecksum bool `json:"validate_checksum"`
	DropInvalid      bool `json:"drop_invalid"`
	PublishRaw       bool `json:"publish_raw"`
}

// DefaultConfig returns default configuration with both processing and port defaults
func DefaultConfig() Config {
	// Component-owned port defaults (not from PortGenerator!)
	return Config{
		// Processing-related defaults
		SystemIDFilter:    []uint8{}, // Accept all
		ComponentIDFilter: []uint8{}, // Accept all
		ProcessHeartbeat:  true,
		ProcessPosition:   true,
		ProcessBattery:    true,
		ProcessStatus:     true,
		ProcessAttitude:   true,
		ProcessGPS:        true,
		ValidateChecksum:  true,
		DropInvalid:       true,
		PublishRaw:        false,

		// Port defaults defined by component (matches constructor)
		Ports: &component.PortConfig{
			Inputs: []component.PortDefinition{
				{
					Name:        "mavlink_input",
					Type:        "nats",
					Subject:     "input.*.mavlink",
					Required:    true,
					Description: "MAVLink protocol messages from vehicles",
					Interface:   "mavlink.v2",
				},
			},
			Outputs: []component.PortDefinition{
				{
					Name:        "storage_write",
					Type:        "nats",
					Subject:     "storage.objectstore.write",
					Required:    false,
					Description: "Graphable messages to ObjectStore",
					Interface:   "message.Graphable",
				},
			},
		},
	}
}

// RoboticsProcessor implements the Processor interface for autonomous vehicles
type RoboticsProcessor struct {
	*processor.BaseProcessor
	nc *gonats.Conn
	// REMOVED: mavlinkParser - creating per-goroutine to avoid concurrency issues
	// See MAVLINK_PARSER_CONCURRENCY_FIX.md for details
	mu                sync.RWMutex
	enabled           bool
	config            Config
	organization      string // Organization for entity ID generation
	platform          string // Platform for entity ID generation
	subscriptions     []*gonats.Subscription
	logger            *slog.Logger
	startTime         time.Time
	messagesProcessed int64
	errors            int64
	lastMessageTime   time.Time
	// Thread safety fields
	shutdown chan struct{} // Signal shutdown to goroutines
	done     chan struct{} // Signal completion of shutdown
	wg       sync.WaitGroup
	stopped  chan struct{} // Keep for compatibility

	// Prometheus metrics
	metrics *RoboticsMetrics

	// Vehicle tracking for metrics
	knownVehicles map[uint8]time.Time
	vehiclesMu    sync.RWMutex

	// Component-owned port definitions
	inputPorts  []component.Port
	outputPorts []component.Port
}

// Ensure RoboticsProcessor implements LifecycleComponent interface
var _ component.LifecycleComponent = (*RoboticsProcessor)(nil)

// roboticsSchema defines the configuration schema for robotics processor component as static metadata
// This allows schema retrieval without component instantiation (Option 1 pattern)
var roboticsSchema = component.ConfigSchema{
	Properties: map[string]component.PropertySchema{
		"process_heartbeat": {
			Type:        "bool",
			Description: "Process MAVLink heartbeat messages",
			Default:     true,
			Category:    "basic",
		},
		"process_position": {
			Type:        "bool",
			Description: "Process MAVLink position messages",
			Default:     true,
			Category:    "basic",
		},
		"process_battery": {
			Type:        "bool",
			Description: "Process MAVLink battery status messages",
			Default:     true,
			Category:    "basic",
		},
		"process_status": {
			Type:        "bool",
			Description: "Process MAVLink system status messages",
			Default:     true,
			Category:    "basic",
		},
		"process_attitude": {
			Type:        "bool",
			Description: "Process MAVLink attitude messages (roll, pitch, yaw)",
			Default:     true,
			Category:    "basic",
		},
		"process_gps": {
			Type:        "bool",
			Description: "Process MAVLink GPS messages",
			Default:     true,
			Category:    "basic",
		},
		"validate_checksum": {
			Type:        "bool",
			Description: "Validate MAVLink message checksums",
			Default:     true,
			Category:    "advanced",
		},
		"drop_invalid": {
			Type:        "bool",
			Description: "Drop messages with invalid checksums",
			Default:     true,
			Category:    "advanced",
		},
		"publish_raw": {
			Type:        "bool",
			Description: "Publish raw MAVLink messages (in addition to semantic messages)",
			Default:     false,
			Category:    "advanced",
		},
	},
	Required: []string{},
}

// RoboticsMetrics holds Prometheus metrics for RoboticsProcessor component
type RoboticsMetrics struct {
	messagesReceived     *prometheus.CounterVec
	messagesProcessed    *prometheus.CounterVec
	messagesDropped      *prometheus.CounterVec
	parseDuration        *prometheus.HistogramVec
	publishDuration      *prometheus.HistogramVec
	checksumErrors       *prometheus.CounterVec
	unknownMessages      *prometheus.CounterVec
	payloadSize          *prometheus.HistogramVec
	activeVehicles       prometheus.Gauge
	lastMessageTimestamp *prometheus.GaugeVec
}

// newRoboticsMetrics creates and registers RoboticsProcessor metrics
func newRoboticsMetrics(registry *metric.MetricsRegistry, instanceName string) *RoboticsMetrics {
	// Return nil if no registry provided (nil input = nil feature pattern)
	if registry == nil {
		return nil
	}

	// Only create metrics when registry is provided
	metrics := &RoboticsMetrics{
		messagesReceived: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "semstreams",
			Subsystem: "robotics",
			Name:      "messages_received_total",
			Help:      "Total raw messages received from NATS",
		}, []string{"subject", "format"}),
		messagesProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "semstreams",
			Subsystem: "robotics",
			Name:      "messages_processed_total",
			Help:      "Messages successfully parsed and published",
		}, []string{"system_id", "message_type", "format"}),
		messagesDropped: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "semstreams",
			Subsystem: "robotics",
			Name:      "messages_dropped_total",
			Help:      "Messages dropped due to parsing errors",
		}, []string{"system_id", "error_type", "format"}),
		parseDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "semstreams",
			Subsystem: "robotics",
			Name:      "parse_duration_seconds",
			Help:      "Time to parse individual messages",
			Buckets:   []float64{0.0001, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05},
		}, []string{"format", "message_type"}),
		publishDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "semstreams",
			Subsystem: "robotics",
			Name:      "publish_duration_seconds",
			Help:      "Time to publish semantic messages to NATS",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
		}, []string{"subject", "payload_type"}),
		checksumErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "semstreams",
			Subsystem: "robotics",
			Name:      "checksum_errors_total",
			Help:      "MAVLink messages with invalid checksums",
		}, []string{"system_id", "message_id"}),
		unknownMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "semstreams",
			Subsystem: "robotics",
			Name:      "unknown_messages_total",
			Help:      "MAVLink messages with unknown message IDs",
		}, []string{"system_id", "message_id"}),
		payloadSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "semstreams",
			Subsystem: "robotics",
			Name:      "payload_size_bytes",
			Help:      "Size distribution of parsed payloads",
			Buckets:   []float64{10, 50, 100, 250, 500, 1000, 2000, 5000},
		}, []string{"payload_type"}),
		activeVehicles: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "semstreams",
			Subsystem: "robotics",
			Name:      "active_vehicles",
			Help:      "Number of unique vehicles recently seen",
		}),
		lastMessageTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "semstreams",
			Subsystem: "robotics",
			Name:      "last_message_timestamp",
			Help:      "Unix timestamp of last processed message",
		}, []string{"system_id", "message_type"}),
	}

	// Register all metrics (no conditional needed since metrics only exist with registry)
	registry.PrometheusRegistry().MustRegister(
		metrics.messagesReceived,
		metrics.messagesProcessed,
		metrics.messagesDropped,
		metrics.parseDuration,
		metrics.publishDuration,
		metrics.checksumErrors,
		metrics.unknownMessages,
		metrics.payloadSize,
		metrics.activeVehicles,
		metrics.lastMessageTimestamp,
	)

	return metrics
}

// NewRoboticsProcessor creates a new robotics processor with required dependencies.
//
// Returns:
//   - ErrNoConnection if NATS connection is nil
//   - Other errors from component initialization
func NewRoboticsProcessor(nc *gonats.Conn) (*RoboticsProcessor, error) {
	return NewRoboticsProcessorWithMetrics(nc, nil)
}

// NewRoboticsProcessorWithMetrics creates a new robotics processor with optional metrics
func NewRoboticsProcessorWithMetrics(nc *gonats.Conn, metricsRegistry *metric.MetricsRegistry) (*RoboticsProcessor, error) {
	if nc == nil {
		return nil, errors.WrapFatal(errors.ErrNoConnection, "RoboticsProcessor", "NewRoboticsProcessor", "validate NATS connection")
	}

	info := processor.ProcessorInfo{
		Name:        "robotics",
		Domain:      "robotics",
		Version:     "1.0.0",
		Description: "Autonomous vehicle robotics processor supporting MAVLink protocol for drones, ground vehicles, boats, and submarines",
		Author:      "SemStreams Robotics Team",
		License:     "MIT",
	}

	baseProcessor := processor.NewBaseProcessor(info)

	return &RoboticsProcessor{
		BaseProcessor: baseProcessor,
		nc:            nc,
		// Parser now created per-message to avoid concurrency issues
		config:          DefaultConfig(),
		startTime:       time.Now(),
		lastMessageTime: time.Now(),
		stopped:         make(chan struct{}),
		metrics:         newRoboticsMetrics(metricsRegistry, "robotics"),
		knownVehicles:   make(map[uint8]time.Time),
		logger:          slog.Default().With("component", "robotics-processor"),
		// Component-owned port definitions (not from PortGenerator!)
		inputPorts: []component.Port{
			{
				Name:        "mavlink_input",
				Direction:   component.DirectionInput,
				Required:    true,
				Description: "MAVLink protocol messages from vehicles",
				Config: component.NATSPort{
					Subject: "input.*.mavlink",
					Interface: &component.InterfaceContract{
						Type:    "mavlink.v2",
						Version: "v1",
					},
				},
			},
		},
		outputPorts: []component.Port{
			{
				Name:        "storage_write",
				Direction:   component.DirectionOutput,
				Required:    false,
				Description: "Graphable messages to ObjectStore",
				Config: component.NATSPort{
					Subject: "storage.objectstore.write",
					Interface: &component.InterfaceContract{
						Type:    "message.Graphable",
						Version: "v1",
					},
				},
			},
		},
	}, nil
}

// Discoverable interface implementation

// Meta returns basic component metadata
func (rp *RoboticsProcessor) Meta() component.Metadata {
	return component.Metadata{
		Name:        "robotics",
		Type:        "processor",
		Description: "Autonomous vehicle robotics processor supporting MAVLink protocol for drones, ground vehicles, boats, and submarines",
		Version:     "1.0.0",
	}
}

// InputPorts returns the input ports for this component
func (rp *RoboticsProcessor) InputPorts() []component.Port {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	// Return component-owned defaults (no PortGenerator!)
	defaults := rp.inputPorts

	// Apply any configured overrides
	if rp.config.Ports != nil && len(rp.config.Ports.Inputs) > 0 {
		return component.MergePortConfigs(defaults, rp.config.Ports.Inputs, component.DirectionInput)
	}

	return defaults
}

// OutputPorts returns the output ports for this component
func (rp *RoboticsProcessor) OutputPorts() []component.Port {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	// Return component-owned defaults (no PortGenerator!)
	defaults := rp.outputPorts

	// Apply any configured overrides
	if rp.config.Ports != nil && len(rp.config.Ports.Outputs) > 0 {
		return component.MergePortConfigs(defaults, rp.config.Ports.Outputs, component.DirectionOutput)
	}

	return defaults
}

// ConfigSchema returns the configuration schema for this component
func (rp *RoboticsProcessor) ConfigSchema() component.ConfigSchema {
	return roboticsSchema
}

// Initialize initializes the robotics processor (setup/create only, NO context)
func (rp *RoboticsProcessor) Initialize() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.enabled = true
	rp.startTime = time.Now()

	// Parser now created per-message to avoid concurrency issues

	return nil
}

// Start starts the robotics processor and sets up NATS subscriptions.
//
// Returns:
//   - ErrAlreadyStarted if processor is already running
//   - ErrSubscriptionFailed if unable to subscribe to NATS subjects
//   - Context errors if context is cancelled during startup
func (rp *RoboticsProcessor) Start(ctx context.Context) error {
	// Get input ports before acquiring lock to avoid deadlock
	inputPorts := rp.InputPorts()

	rp.mu.Lock()
	defer rp.mu.Unlock()

	// Check if already started
	if rp.shutdown != nil {
		return errors.WrapInvalid(errors.ErrAlreadyStarted, "RoboticsProcessor", "Start", "check processor state")
	}

	// Create shutdown channels for coordinated shutdown
	rp.shutdown = make(chan struct{})
	rp.done = make(chan struct{})
	for _, port := range inputPorts {
		// Extract subject from port config
		var subject string
		switch cfg := port.Config.(type) {
		case component.NATSPort:
			subject = cfg.Subject
		default:
			continue // Skip non-NATS ports
		}

		subscription, err := rp.nc.Subscribe(subject, func(msg *gonats.Msg) {
			// Process messages asynchronously to avoid blocking NATS
			rp.wg.Add(1)
			go func() {
				defer rp.wg.Done()

				select {
				case <-rp.shutdown:
					return // Shutdown signal received, exit goroutine
				default:
					// Use background context for NATS callback handlers
					if err := rp.ProcessRawData(context.Background(), msg.Subject, msg.Data); err != nil {
						rp.logger.Error("Error processing message", "subject", msg.Subject, "error", err)
						rp.mu.Lock()
						rp.errors++
						rp.mu.Unlock()
					} else {
						rp.mu.Lock()
						rp.messagesProcessed++
						rp.lastMessageTime = time.Now()
						rp.mu.Unlock()
					}
				}
			}()
		})
		if err != nil {
			// Clean up channels on error
			if rp.shutdown != nil {
				close(rp.shutdown)
				rp.shutdown = nil
				rp.done = nil
			}
			return errors.Wrap(err, "RoboticsProcessor", "Start", fmt.Sprintf("subscribe to %s", subject))
		}

		// Set subscription pending limits to handle concurrent load
		if err := subscription.SetPendingLimits(5000, 50*1024*1024); err != nil {
			rp.logger.Warn("Could not set pending limits", "subject", subject, "error", err)
		}

		rp.subscriptions = append(rp.subscriptions, subscription)
		rp.logger.Info("Subscribed to NATS subject", "subject", subject)
	}

	rp.logger.Info("Robotics processor started", "subscription_count", len(rp.subscriptions))
	return nil
}

// Stop stops the robotics processor and cleans up resources with timeout
func (rp *RoboticsProcessor) Stop(timeout time.Duration) error {
	rp.mu.Lock()

	// Check if already stopped
	if rp.shutdown == nil {
		rp.mu.Unlock()
		return nil // Already stopped
	}

	// Signal shutdown to all goroutines (safe to close once)
	select {
	case <-rp.shutdown:
		// Already closed
	default:
		close(rp.shutdown)
	}

	// Unsubscribe from all subscriptions
	subscriptionsCount := len(rp.subscriptions)
	for _, sub := range rp.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			rp.logger.Error("Error unsubscribing", "error", err)
		}
	}
	rp.subscriptions = nil

	rp.mu.Unlock()

	// Wait for all goroutines to complete with timeout
	doneChan := make(chan struct{})
	go func() {
		rp.wg.Wait()
		close(doneChan)
	}()

	select {
	case <-doneChan:
		// All goroutines completed gracefully
		rp.logger.Info("Stopped gracefully")
	case <-time.After(timeout):
		// Timeout exceeded - this shouldn't happen with proper shutdown
		rp.logger.Error("Shutdown timeout exceeded", "timeout", timeout)
		return fmt.Errorf("shutdown timeout exceeded after %v", timeout)
	}

	// Signal completion via done channel
	rp.mu.Lock()
	if rp.done != nil {
		select {
		case <-rp.done:
		default:
			close(rp.done)
		}
	}
	rp.mu.Unlock()

	// Signal stopped for compatibility
	select {
	case <-rp.stopped:
	default:
		close(rp.stopped)
	}

	// Reset channels for potential restart
	rp.mu.Lock()
	rp.shutdown = nil
	rp.done = nil
	rp.mu.Unlock()

	rp.logger.Info("Robotics processor stopped", "cleaned_subscriptions", subscriptionsCount)
	return nil
}

// Health returns current health status
func (rp *RoboticsProcessor) Health() component.HealthStatus {
	rp.mu.RLock()
	enabled := rp.enabled
	rp.mu.RUnlock()

	baseHealth := rp.BaseProcessor.Health()

	return component.HealthStatus{
		Healthy:    enabled && baseHealth.Healthy,
		LastCheck:  time.Now(),
		ErrorCount: int(rp.errors),
		LastError:  "",
		Uptime:     time.Since(rp.startTime),
	}
}

// DataFlow returns current data flow metrics
func (rp *RoboticsProcessor) DataFlow() component.FlowMetrics {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	var messagesPerSecond float64
	if uptime := time.Since(rp.startTime).Seconds(); uptime > 0 {
		messagesPerSecond = float64(rp.messagesProcessed) / uptime
	}

	var errorRate float64
	if rp.messagesProcessed > 0 {
		errorRate = float64(rp.errors) / float64(rp.messagesProcessed)
	}

	return component.FlowMetrics{
		MessagesPerSecond: messagesPerSecond,
		BytesPerSecond:    0, // TODO: Track bytes if needed
		ErrorRate:         errorRate,
		LastActivity:      rp.lastMessageTime,
	}
}

// Configuration returns the current robotics processor configuration
func (rp *RoboticsProcessor) Configuration() any {
	rp.mu.RLock()
	enabled := rp.enabled
	rp.mu.RUnlock()

	baseConfig := rp.BaseProcessor.Configuration().(map[string]any)
	baseConfig["enabled"] = enabled
	baseConfig["formats"] = []string{"mavlink", "json"}
	baseConfig["outputs"] = []string{
		"process.robotics.heartbeat",
		"process.robotics.position",
		"process.robotics.attitude",
		"process.robotics.battery",
	}
	return baseConfig
}

// ValidateConfiguration validates robotics processor configuration.
//
// Returns:
//   - ErrInvalidConfig if config is not a map or has invalid fields
func (rp *RoboticsProcessor) ValidateConfiguration(config any) error {
	configMap, ok := config.(map[string]any)
	if !ok {
		return errors.WrapInvalid(errors.ErrInvalidConfig, "RoboticsProcessor", "ValidateConfiguration", "validate config type")
	}

	if enabled, exists := configMap["enabled"]; exists {
		if _, ok := enabled.(bool); !ok {
			return errors.WrapInvalid(errors.ErrInvalidConfig, "RoboticsProcessor", "ValidateConfiguration", "validate enabled field")
		}
	}

	return nil
}

// ReloadConfiguration reloads robotics processor configuration.
//
// Returns:
//   - ErrInvalidConfig if config is not a map
func (rp *RoboticsProcessor) ReloadConfiguration(_ context.Context, config any) error {
	configMap, ok := config.(map[string]any)
	if !ok {
		return errors.WrapInvalid(errors.ErrInvalidConfig, "RoboticsProcessor", "ReloadConfiguration", "validate config type")
	}

	if enabled, exists := configMap["enabled"]; exists {
		if enabledBool, ok := enabled.(bool); ok {
			rp.mu.Lock()
			rp.enabled = enabledBool
			rp.mu.Unlock()
		}
	}

	return nil
}

// ProcessRawData processes raw robotics data from a given subject
func (rp *RoboticsProcessor) ProcessRawData(ctx context.Context, subject string, data []byte) error {
	// Record message received metric
	format := "unknown"
	if strings.Contains(subject, "mavlink") {
		format = "mavlink"
	} else if strings.Contains(subject, "json") {
		format = "json"
	}

	if rp.metrics != nil {
		rp.metrics.messagesReceived.WithLabelValues(subject, format).Inc()
	}

	// Get NATS connection from context or use the one from Initialize
	rp.mu.RLock()
	nc := rp.nc
	enabled := rp.enabled
	rp.mu.RUnlock()

	if !enabled {
		return nil // Skip processing if disabled
	}

	if nc == nil {
		return errors.WrapFatal(errors.ErrNoConnection, "RoboticsProcessor", "ProcessRawData", "check NATS connection")
	}

	// Determine format from subject and process
	if format == "mavlink" {
		return rp.processMAVLinkData(ctx, nc, subject, data)
	} else if format == "json" {
		return rp.processJSONData(ctx, nc, subject, data)
	}

	// Default to JSON processing
	return rp.processJSONData(ctx, nc, subject, data)
}

// Metrics returns robotics processor metrics
func (rp *RoboticsProcessor) Metrics() map[string]any {
	rp.mu.RLock()
	enabled := rp.enabled
	subCount := len(rp.subscriptions)
	rp.mu.RUnlock()

	metrics := rp.BaseProcessor.Metrics()
	metrics["enabled"] = enabled
	metrics["subscription_count"] = subCount
	metrics["supported_formats"] = []string{"mavlink", "json"}
	metrics["vehicle_types"] = []string{"drone", "ground_vehicle", "boat", "submarine"}
	return metrics
}

// SubscriptionPatterns returns the patterns this processor subscribes to
func (rp *RoboticsProcessor) SubscriptionPatterns() []string {
	// Get subjects from input ports
	inputPorts := rp.InputPorts()
	patterns := make([]string, 0, len(inputPorts))
	for _, input := range inputPorts {
		if natsPort, ok := input.Config.(component.NATSPort); ok {
			patterns = append(patterns, natsPort.Subject)
		}
	}
	return patterns
}

// ApplyConfig applies configuration to the processor
func (rp *RoboticsProcessor) ApplyConfig(config Config) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.config = config
}

// Config returns current configuration
func (rp *RoboticsProcessor) Config() Config {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.config
}

// ParseConfig parses configuration from map
func ParseConfig(configData map[string]any) Config {
	// Start with empty config for convention-based defaults
	var config Config
	if configData == nil {
		// Apply only processing defaults when no config provided
		config.ProcessHeartbeat = true
		config.ProcessPosition = true
		config.ProcessBattery = true
		config.ProcessStatus = true
		config.ProcessAttitude = true
		config.ProcessGPS = true
		config.ValidateChecksum = true
		config.DropInvalid = true
		return config
	}

	parseFiltering(&config, configData)
	parseMessageTypes(&config, configData)
	parseProcessingOptions(&config, configData)
	return config
}

// parseFiltering extracts filtering configuration
func parseFiltering(config *Config, data map[string]any) {
	if sysIDs, ok := data["system_id_filter"].([]any); ok {
		config.SystemIDFilter = parseUint8Array(sysIDs)
	}
	if compIDs, ok := data["component_id_filter"].([]any); ok {
		config.ComponentIDFilter = parseUint8Array(compIDs)
	}
}

// parseMessageTypes extracts message type processing flags
func parseMessageTypes(config *Config, data map[string]any) {
	parseBool(data, "process_heartbeat", &config.ProcessHeartbeat)
	parseBool(data, "process_position", &config.ProcessPosition)
	parseBool(data, "process_battery", &config.ProcessBattery)
	parseBool(data, "process_status", &config.ProcessStatus)
	parseBool(data, "process_attitude", &config.ProcessAttitude)
	parseBool(data, "process_gps", &config.ProcessGPS)
}

// parseProcessingOptions extracts processing option flags
func parseProcessingOptions(config *Config, data map[string]any) {
	parseBool(data, "validate_checksum", &config.ValidateChecksum)
	parseBool(data, "drop_invalid", &config.DropInvalid)
	parseBool(data, "publish_raw", &config.PublishRaw)
}

// parseStringArray converts []any to []string, filtering valid strings
func parseStringArray(arr []any) []string {
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if str, ok := v.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// parseUint8Array converts []any to []uint8, filtering valid numbers
func parseUint8Array(arr []any) []uint8 {
	result := make([]uint8, 0, len(arr))
	for _, v := range arr {
		if num, ok := v.(float64); ok {
			result = append(result, uint8(num))
		}
	}
	return result
}

// parseBool safely extracts a boolean value from map data
func parseBool(data map[string]any, key string, target *bool) {
	if val, ok := data[key].(bool); ok {
		*target = val
	}
}

// processMAVLinkData processes MAVLink binary data using the robust parser
func (rp *RoboticsProcessor) processMAVLinkData(ctx context.Context, nc *gonats.Conn, subject string, data []byte) error {
	rp.logger.Debug("Processing MAVLink data", "bytes", len(data), "subject", subject)

	// Create a new parser for this goroutine to avoid concurrency issues
	// Each message handler gets its own parser instance with no shared state
	mavlinkParser := parser.NewMAVLinkParser()

	// Parse MAVLink packets using the robust parser
	packets, err := mavlinkParser.Parse(data)
	if err != nil {
		rp.logger.Warn("MAVLink parse error", "error", err)
		// Don't return error - keep processing other data
		return nil
	}

	rp.logger.Debug("Parsed MAVLink packets", "count", len(packets))

	// Get configuration for filtering
	rp.mu.RLock()
	config := rp.config
	rp.mu.RUnlock()

	// Convert each packet to semantic messages
	for _, packet := range packets {
		if err := rp.processMAVLinkPacket(ctx, nc, packet, config, mavlinkParser); err != nil {
			rp.logger.Error("Error processing packet", "message_id", packet.MessageID, "error", err)
		}
	}

	return nil
}

// processMAVLinkPacket processes a single MAVLink packet
func (rp *RoboticsProcessor) processMAVLinkPacket(ctx context.Context, nc *gonats.Conn, packet *parser.MAVLinkPacket, config Config, mavlinkParser *parser.MAVLinkParser) error {
	rp.logger.Debug("Processing packet", "system_id", packet.SystemID, "message_id", packet.MessageID)

	// Track active vehicles for metrics
	rp.trackVehicleActivity(packet.SystemID)

	// Measure parse duration
	start := time.Now()

	// Apply filters
	if !rp.passesSystemIDFilter(packet.SystemID, config) {
		rp.logger.Debug("Skipping packet from system ID (filtered)", "system_id", packet.SystemID)
		if rp.metrics != nil {
			rp.metrics.messagesDropped.WithLabelValues(fmt.Sprintf("%d", packet.SystemID), "filtered", "mavlink").Inc()
		}
		return nil
	}

	if !rp.passesComponentIDFilter(packet.ComponentID, config) {
		rp.logger.Debug("Skipping packet from component ID (filtered)", "component_id", packet.ComponentID)
		if rp.metrics != nil {
			rp.metrics.messagesDropped.WithLabelValues(fmt.Sprintf("%d", packet.SystemID), "filtered", "mavlink").Inc()
		}
		return nil
	}

	// Check if message type is enabled
	if !rp.isMessageTypeEnabled(packet.MessageID) {
		rp.logger.Debug("Skipping message ID (disabled)", "message_id", packet.MessageID)
		if rp.metrics != nil {
			rp.metrics.messagesDropped.WithLabelValues(fmt.Sprintf("%d", packet.SystemID), "disabled", "mavlink").Inc()
		}
		return nil
	}

	messageType := rp.getMessageTypeName(packet.MessageID)

	// Record parse duration
	if rp.metrics != nil {
		duration := time.Since(start)
		rp.metrics.parseDuration.WithLabelValues("mavlink", messageType).Observe(duration.Seconds())
	}

	return rp.convertAndPublishPacket(ctx, nc, packet, config, mavlinkParser)
}

// trackVehicleActivity tracks vehicle activity for metrics
func (rp *RoboticsProcessor) trackVehicleActivity(systemID uint8) {
	now := time.Now()

	rp.vehiclesMu.Lock()
	defer rp.vehiclesMu.Unlock()

	rp.knownVehicles[systemID] = now

	// Update metrics
	if rp.metrics != nil {
		// Clean up vehicles not seen in the last 5 minutes
		cutoff := now.Add(-5 * time.Minute)
		activeCount := 0
		for id, lastSeen := range rp.knownVehicles {
			if lastSeen.After(cutoff) {
				activeCount++
			} else {
				delete(rp.knownVehicles, id)
			}
		}

		rp.metrics.activeVehicles.Set(float64(activeCount))
	}
}

// passesSystemIDFilter checks if packet passes system ID filter
func (rp *RoboticsProcessor) passesSystemIDFilter(systemID uint8, config Config) bool {
	if len(config.SystemIDFilter) == 0 {
		return true // No filter = allow all
	}

	for _, allowedID := range config.SystemIDFilter {
		if systemID == allowedID {
			return true
		}
	}
	return false
}

// passesComponentIDFilter checks if packet passes component ID filter
func (rp *RoboticsProcessor) passesComponentIDFilter(componentID uint8, config Config) bool {
	if len(config.ComponentIDFilter) == 0 {
		return true // No filter = allow all
	}

	for _, allowedID := range config.ComponentIDFilter {
		if componentID == allowedID {
			return true
		}
	}
	return false
}

// convertAndPublishPacket converts packet to SemStreams message and publishes it
func (rp *RoboticsProcessor) convertAndPublishPacket(_ context.Context, nc *gonats.Conn, packet *parser.MAVLinkPacket, config Config, mavlinkParser *parser.MAVLinkParser) error {
	systemIDStr := fmt.Sprintf("%d", packet.SystemID)
	messageType := rp.getMessageTypeName(packet.MessageID)

	// Convert to SemStreams message using the parser's conversion
	msg, err := mavlinkParser.ConvertToSemStreamsMessage(packet)
	if err != nil {
		rp.logger.Error("Conversion error for message ID", "message_id", packet.MessageID, "error", err)
		if rp.metrics != nil {
			rp.metrics.messagesDropped.WithLabelValues(systemIDStr, "conversion_error", "mavlink").Inc()
		}
		if config.DropInvalid {
			return nil
		}
	}

	// Create Type based on packet type
	msgType := messages.Type{
		Domain:   "robotics",
		Category: rp.getMessageCategory(packet.MessageID),
		Version:  "v1",
	}

	// Create BaseMessage wrapper with the payload from parser
	payload := msg.Payload()
	baseMsg := messages.NewBaseMessage(msgType, payload, "robotics-processor")

	rp.logger.Debug("Created BaseMessage", "type", msgType.String(), "payload", payload)

	// Marshal the complete BaseMessage (not just payload!)
	msgBytes, err := json.Marshal(baseMsg)
	if err != nil {
		rp.logger.Error("Marshal error", "error", err)
		if rp.metrics != nil {
			rp.metrics.messagesDropped.WithLabelValues(systemIDStr, "marshal_error", "mavlink").Inc()
		}
		return errors.Wrap(err, "RoboticsProcessor", "convertAndPublishPacket", "marshal BaseMessage")
	}

	// Record payload size metric
	if rp.metrics != nil {
		payloadType := msgType.Category
		rp.metrics.payloadSize.WithLabelValues(payloadType).Observe(float64(len(msgBytes)))
	}

	// Determine the output subject based on message type
	outputSubject := rp.getOutputSubject(packet.MessageID)
	rp.logger.Debug("Publishing BaseMessage", "subject", outputSubject)

	// Measure publish duration
	start := time.Now()

	// Publish BaseMessage format
	if err := nc.Publish(outputSubject, msgBytes); err != nil {
		rp.logger.Error("Publish error", "subject", outputSubject, "error", err)
		if rp.metrics != nil {
			rp.metrics.messagesDropped.WithLabelValues(systemIDStr, "publish_error", "mavlink").Inc()
		}
		return errors.WrapTransient(err, "RoboticsProcessor", "convertAndPublishPacket",
			fmt.Sprintf("publish to %s", outputSubject))
	}

	// Record successful metrics
	if rp.metrics != nil {
		// Record publish duration
		duration := time.Since(start)
		rp.metrics.publishDuration.WithLabelValues(outputSubject, msgType.Category).Observe(duration.Seconds())

		// Record successful processing
		rp.metrics.messagesProcessed.WithLabelValues(systemIDStr, messageType, "mavlink").Inc()

		// Record last message timestamp
		rp.metrics.lastMessageTimestamp.WithLabelValues(systemIDStr, messageType).Set(float64(time.Now().Unix()))
	}

	rp.logger.Debug("Successfully published BaseMessage", "message_type", rp.getMessageTypeName(packet.MessageID), "system_id", packet.SystemID)
	return nil
}

// processJSONData processes JSON-formatted robotics data
func (rp *RoboticsProcessor) processJSONData(_ context.Context, nc *gonats.Conn, _ string, data []byte) error {
	var parsedData map[string]any
	if err := json.Unmarshal(data, &parsedData); err != nil {
		return errors.WrapInvalid(err, "RoboticsProcessor", "processJSONData", "parse JSON data")
	}

	// Determine message type from data structure
	if _, hasSystemID := parsedData["system_id"]; hasSystemID {
		if _, hasHeartbeat := parsedData["type"]; hasHeartbeat {
			// Heartbeat message
			return rp.publishHeartbeatMessage(context.Background(), nc, parsedData)
		}
		if _, hasLat := parsedData["latitude"]; hasLat {
			// Position message
			return rp.publishPositionMessage(context.Background(), nc, parsedData)
		}
		if _, hasRoll := parsedData["roll"]; hasRoll {
			// Attitude message
			return rp.publishAttitudeMessage(context.Background(), nc, parsedData)
		}
		if _, hasBattery := parsedData["battery_remaining"]; hasBattery {
			// Battery message
			return rp.publishBatteryMessage(context.Background(), nc, parsedData)
		}
	}

	return errors.WrapInvalid(errors.ErrInvalidData, "RoboticsProcessor", "processJSONData", "identify message type")
}

// processHeartbeat processes MAVLink heartbeat messages.
//
// Returns:
//   - ErrInvalidData for malformed heartbeat data
//   - Marshaling errors for message serialization failures
func (rp *RoboticsProcessor) processHeartbeat(_ context.Context, nc *gonats.Conn, data []byte) error {
	// Simplified MAVLink heartbeat parsing
	// Real implementation would use proper MAVLink library
	systemID := uint8(data[3]) // System ID at byte 3

	// Create heartbeat message
	heartbeat := payloads.NewHeartbeatPayload(systemID, 0, time.Now())

	// Wrap in Message
	msg := messages.NewBaseMessage(
		heartbeat.Schema(),
		heartbeat,
		"robotics-processor",
	)

	// Marshal the MESSAGE
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "RoboticsProcessor", "processHeartbeat", "marshal heartbeat message")
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(0) // HEARTBEAT
	rp.mu.RUnlock()
	return nc.Publish(subject, msgBytes)
}

// processGlobalPosition processes MAVLink global position messages
func (rp *RoboticsProcessor) processGlobalPosition(_ context.Context, nc *gonats.Conn, data []byte) error {
	// Simplified MAVLink position parsing
	systemID := uint8(data[3])

	// Real implementation would parse lat/lon from MAVLink payload
	lat := 0.0 // Would be parsed from bytes 6-9
	lon := 0.0 // Would be parsed from bytes 10-13

	// Create position message
	position := payloads.NewPositionPayload(systemID, time.Now(), lat, lon)

	// Wrap in Message
	msg := messages.NewBaseMessage(
		position.Schema(),
		position,
		"robotics-processor",
	)

	// Marshal the MESSAGE
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "RoboticsProcessor", "marshal", "marshal position message")
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(33) // GLOBAL_POSITION_INT
	rp.mu.RUnlock()
	return nc.Publish(subject, msgBytes)
}

// processAttitude processes MAVLink attitude messages
func (rp *RoboticsProcessor) processAttitude(_ context.Context, nc *gonats.Conn, data []byte) error {
	// Simplified MAVLink attitude parsing
	systemID := uint8(data[3])

	// Create attitude message
	attitude := payloads.NewAttitudePayload(systemID, time.Now())

	// Wrap in Message
	msg := messages.NewBaseMessage(
		attitude.Schema(),
		attitude,
		"robotics-processor",
	)

	// Marshal the MESSAGE
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "RoboticsProcessor", "marshal", "marshal attitude message")
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(30) // ATTITUDE
	rp.mu.RUnlock()
	return nc.Publish(subject, msgBytes)
}

// processBatteryStatus processes MAVLink battery status messages
func (rp *RoboticsProcessor) processBatteryStatus(_ context.Context, nc *gonats.Conn, data []byte) error {
	// Simplified MAVLink battery parsing
	systemID := uint8(data[3])
	batteryID := uint8(0) // Would be parsed from payload

	// Create battery message
	battery := payloads.NewBatteryPayload(systemID, batteryID, time.Now())

	// Wrap in Message
	msg := messages.NewBaseMessage(
		battery.Schema(),
		battery,
		"robotics-processor",
	)

	// Marshal the MESSAGE
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "RoboticsProcessor", "marshal", "marshal battery message")
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(147) // BATTERY_STATUS
	rp.mu.RUnlock()
	return nc.Publish(subject, msgBytes)
}

// publishHeartbeatMessage creates and publishes a heartbeat message from JSON
func (rp *RoboticsProcessor) publishHeartbeatMessage(_ context.Context, nc *gonats.Conn, data map[string]any) error {
	systemID, ok := data["system_id"].(float64)
	if !ok {
		return errors.WrapInvalid(errors.ErrInvalidData, "RoboticsProcessor", "validate", "validate system_id")
	}

	componentID := uint8(0)
	if cid, ok := data["component_id"].(float64); ok {
		componentID = uint8(cid)
	}

	// Create and publish heartbeat message
	payload := payloads.NewHeartbeatPayload(uint8(systemID), componentID, time.Now())

	// Set optional fields
	if vehicleType, ok := data["type"].(float64); ok {
		payload.VehicleType = uint8(vehicleType)
	}

	// Wrap in Message
	msg := messages.NewBaseMessage(
		payload.Schema(),
		payload,
		"robotics-processor",
	)

	// Marshal the MESSAGE
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "RoboticsProcessor", "marshal", "marshal heartbeat message")
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(0) // HEARTBEAT
	rp.mu.RUnlock()
	return nc.Publish(subject, msgBytes)
}

// publishPositionMessage creates and publishes a position message from JSON
func (rp *RoboticsProcessor) publishPositionMessage(_ context.Context, nc *gonats.Conn, data map[string]any) error {
	systemID, ok := data["system_id"].(float64)
	if !ok {
		return errors.WrapInvalid(errors.ErrInvalidData, "RoboticsProcessor", "validate", "validate system_id")
	}

	lat, ok := data["latitude"].(float64)
	if !ok {
		return errors.WrapInvalid(errors.ErrInvalidData, "RoboticsProcessor", "publishPositionMessage", "validate latitude")
	}

	lon, ok := data["longitude"].(float64)
	if !ok {
		return errors.WrapInvalid(errors.ErrInvalidData, "RoboticsProcessor", "publishPositionMessage", "validate longitude")
	}

	// Create and publish position message
	position := payloads.NewPositionPayload(uint8(systemID), time.Now(), lat, lon)

	// Wrap in Message
	msg := messages.NewBaseMessage(
		position.Schema(),
		position,
		"robotics-processor",
	)

	// Marshal the MESSAGE
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "RoboticsProcessor", "marshal", "marshal position message")
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(33) // GLOBAL_POSITION_INT
	rp.mu.RUnlock()
	return nc.Publish(subject, msgBytes)
}

// publishAttitudeMessage creates and publishes an attitude message from JSON
func (rp *RoboticsProcessor) publishAttitudeMessage(_ context.Context, nc *gonats.Conn, data map[string]any) error {
	systemID, ok := data["system_id"].(float64)
	if !ok {
		return errors.WrapInvalid(errors.ErrInvalidData, "RoboticsProcessor", "validate", "validate system_id")
	}

	// Create and publish attitude message
	payload := payloads.NewAttitudePayload(uint8(systemID), time.Now())

	// Set optional fields
	if roll, ok := data["roll"].(float64); ok {
		payload.Roll = float32(roll)
	}
	if pitch, ok := data["pitch"].(float64); ok {
		payload.Pitch = float32(pitch)
	}
	if yaw, ok := data["yaw"].(float64); ok {
		payload.Yaw = float32(yaw)
	}

	// Wrap in Message
	msg := messages.NewBaseMessage(
		payload.Schema(),
		payload,
		"robotics-processor",
	)

	// Marshal the MESSAGE
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "RoboticsProcessor", "marshal", "marshal attitude message")
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(30) // ATTITUDE
	rp.mu.RUnlock()
	return nc.Publish(subject, msgBytes)
}

// publishBatteryMessage creates and publishes a battery message from JSON
func (rp *RoboticsProcessor) publishBatteryMessage(_ context.Context, nc *gonats.Conn, data map[string]any) error {
	systemID, ok := data["system_id"].(float64)
	if !ok {
		return errors.WrapInvalid(errors.ErrInvalidData, "RoboticsProcessor", "validate", "validate system_id")
	}

	batteryID := uint8(0)
	if bid, ok := data["battery_id"].(float64); ok {
		batteryID = uint8(bid)
	}

	// Create and publish battery message
	payload := payloads.NewBatteryPayload(uint8(systemID), batteryID, time.Now())

	// Set optional fields
	if remaining, ok := data["battery_remaining"].(float64); ok {
		payload.BatteryRemaining = int8(remaining)
	}
	if voltage, ok := data["voltage"].(float64); ok {
		payload.TotalVoltage = voltage
	}

	// Wrap in Message
	msg := messages.NewBaseMessage(
		payload.Schema(),
		payload,
		"robotics-processor",
	)

	// Marshal the MESSAGE
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "RoboticsProcessor", "marshal", "marshal battery message")
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(147) // BATTERY_STATUS
	rp.mu.RUnlock()
	return nc.Publish(subject, msgBytes)
}

// GetNATSConnection returns the current NATS connection
func (rp *RoboticsProcessor) GetNATSConnection() *gonats.Conn {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.nc
}

// handleMAVLinkMessage processes MAVLink messages directly
func (rp *RoboticsProcessor) handleMAVLinkMessage(msg *gonats.Msg) {
	rp.mu.RLock()
	enabled := rp.enabled
	rp.mu.RUnlock()

	if !enabled {
		return
	}

	// Extract raw data from envelope
	var envelope map[string]any
	if err := json.Unmarshal(msg.Data, &envelope); err != nil {
		rp.logger.Error("Failed to unmarshal envelope", "error", err)
		return
	}

	rawData, ok := envelope["raw"].(string)
	if !ok {
		rp.logger.Warn("No 'raw' field found in envelope")
		return
	}

	data := []byte(rawData)
	rp.processDirectMAVLinkData(data)
}

// handleJSONMessage processes JSON robotics data
func (rp *RoboticsProcessor) handleJSONMessage(msg *gonats.Msg) {
	rp.mu.RLock()
	enabled := rp.enabled
	rp.mu.RUnlock()

	if !enabled {
		return
	}

	// Extract raw data from envelope
	var envelope map[string]any
	if err := json.Unmarshal(msg.Data, &envelope); err != nil {
		rp.logger.Error("Failed to unmarshal JSON envelope", "error", err)
		return
	}

	rawData, ok := envelope["raw"].(string)
	if !ok {
		rp.logger.Warn("No 'raw' field found in JSON envelope")
		return
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(rawData), &data); err != nil {
		return
	}

	// Route based on data content
	if _, hasSystemID := data["system_id"]; hasSystemID {
		if _, hasHeartbeat := data["type"]; hasHeartbeat {
			rp.publishHeartbeatFromJSON(data)
		} else if _, hasLat := data["latitude"]; hasLat {
			rp.publishPositionFromJSON(data)
		} else if _, hasRoll := data["roll"]; hasRoll {
			rp.publishAttitudeFromJSON(data)
		} else if _, hasBattery := data["battery_remaining"]; hasBattery {
			rp.publishBatteryFromJSON(data)
		}
	}
}

// processDirectMAVLinkData processes MAVLink binary data directly using robust parser
func (rp *RoboticsProcessor) processDirectMAVLinkData(data []byte) {
	nc := rp.GetNATSConnection()
	if nc == nil {
		rp.logger.Warn("No NATS connection available for direct MAVLink processing")
		return
	}

	// Use the robust parser
	if err := rp.processMAVLinkData(context.Background(), nc, "direct", data); err != nil {
		rp.logger.Error("Error processing direct MAVLink data", "error", err)
	}
}

// publishHeartbeat processes MAVLink heartbeat data and publishes the message
func (rp *RoboticsProcessor) publishHeartbeat(data []byte) {
	nc := rp.GetNATSConnection()
	if nc == nil {
		return
	}

	// Simplified MAVLink heartbeat parsing
	systemID := uint8(data[3]) // System ID at byte 3

	// Create heartbeat message
	heartbeat := payloads.NewHeartbeatPayload(systemID, 0, time.Now())

	// Wrap in Message
	msg := messages.NewBaseMessage(
		heartbeat.Schema(),
		heartbeat,
		"robotics-processor",
	)

	// Marshal and publish
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		rp.logger.Error("Error marshaling heartbeat message", "error", err)
		return
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(0) // HEARTBEAT
	rp.mu.RUnlock()
	nc.Publish(subject, msgBytes)
}

// publishPosition processes MAVLink position data and publishes the message
func (rp *RoboticsProcessor) publishPosition(data []byte) {
	nc := rp.GetNATSConnection()
	if nc == nil {
		return
	}

	// Simplified MAVLink position parsing
	systemID := uint8(data[3])

	// Real implementation would parse lat/lon from MAVLink payload
	lat := 0.0 // Would be parsed from bytes 6-9
	lon := 0.0 // Would be parsed from bytes 10-13

	// Create position message
	position := payloads.NewPositionPayload(systemID, time.Now(), lat, lon)

	// Wrap in Message
	msg := messages.NewBaseMessage(
		position.Schema(),
		position,
		"robotics-processor",
	)

	// Marshal and publish
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		rp.logger.Error("Error marshaling position message", "error", err)
		return
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(33) // GLOBAL_POSITION_INT
	rp.mu.RUnlock()
	nc.Publish(subject, msgBytes)
}

// publishAttitude processes MAVLink attitude data and publishes the message
func (rp *RoboticsProcessor) publishAttitude(data []byte) {
	nc := rp.GetNATSConnection()
	if nc == nil {
		return
	}

	// Simplified MAVLink attitude parsing
	systemID := uint8(data[3])

	// Create attitude message
	attitude := payloads.NewAttitudePayload(systemID, time.Now())

	// Wrap in Message
	msg := messages.NewBaseMessage(
		attitude.Schema(),
		attitude,
		"robotics-processor",
	)

	// Marshal and publish
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		rp.logger.Error("Error marshaling attitude message", "error", err)
		return
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(30) // ATTITUDE
	rp.mu.RUnlock()
	nc.Publish(subject, msgBytes)
}

// publishBattery processes MAVLink battery data and publishes the message
func (rp *RoboticsProcessor) publishBattery(data []byte) {
	nc := rp.GetNATSConnection()
	if nc == nil {
		return
	}

	// Simplified MAVLink battery parsing
	systemID := uint8(data[3])
	batteryID := uint8(0) // Would be parsed from payload

	// Create battery message
	battery := payloads.NewBatteryPayload(systemID, batteryID, time.Now())

	// Wrap in Message
	msg := messages.NewBaseMessage(
		battery.Schema(),
		battery,
		"robotics-processor",
	)

	// Marshal and publish
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		rp.logger.Error("Error marshaling battery message", "error", err)
		return
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(147) // BATTERY_STATUS
	rp.mu.RUnlock()
	nc.Publish(subject, msgBytes)
}

// publishHeartbeatFromJSON creates and publishes a heartbeat message from JSON
func (rp *RoboticsProcessor) publishHeartbeatFromJSON(data map[string]any) {
	nc := rp.GetNATSConnection()
	if nc == nil {
		return
	}

	systemID, ok := data["system_id"].(float64)
	if !ok {
		return
	}

	componentID := uint8(0)
	if cid, ok := data["component_id"].(float64); ok {
		componentID = uint8(cid)
	}

	// Create heartbeat payload
	payload := payloads.NewHeartbeatPayload(uint8(systemID), componentID, time.Now())

	// Set optional fields
	if vehicleType, ok := data["type"].(float64); ok {
		payload.VehicleType = uint8(vehicleType)
	}

	// Wrap in Message
	msg := messages.NewBaseMessage(
		payload.Schema(),
		payload,
		"robotics-processor",
	)

	// Marshal and publish
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		rp.logger.Error("Error marshaling heartbeat message", "error", err)
		return
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(0) // HEARTBEAT
	rp.mu.RUnlock()
	nc.Publish(subject, msgBytes)
}

// publishPositionFromJSON creates and publishes a position message from JSON
func (rp *RoboticsProcessor) publishPositionFromJSON(data map[string]any) {
	nc := rp.GetNATSConnection()
	if nc == nil {
		return
	}

	systemID, ok := data["system_id"].(float64)
	if !ok {
		return
	}

	lat, ok := data["latitude"].(float64)
	if !ok {
		return
	}

	lon, ok := data["longitude"].(float64)
	if !ok {
		return
	}

	// Create position message
	position := payloads.NewPositionPayload(uint8(systemID), time.Now(), lat, lon)

	// Wrap in Message
	msg := messages.NewBaseMessage(
		position.Schema(),
		position,
		"robotics-processor",
	)

	// Marshal and publish
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		rp.logger.Error("Error marshaling position message", "error", err)
		return
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(33) // GLOBAL_POSITION_INT
	rp.mu.RUnlock()
	nc.Publish(subject, msgBytes)
}

// publishAttitudeFromJSON creates and publishes an attitude message from JSON
func (rp *RoboticsProcessor) publishAttitudeFromJSON(data map[string]any) {
	nc := rp.GetNATSConnection()
	if nc == nil {
		return
	}

	systemID, ok := data["system_id"].(float64)
	if !ok {
		return
	}

	// Create attitude message
	payload := payloads.NewAttitudePayload(uint8(systemID), time.Now())

	// Set optional fields
	if roll, ok := data["roll"].(float64); ok {
		payload.Roll = float32(roll)
	}
	if pitch, ok := data["pitch"].(float64); ok {
		payload.Pitch = float32(pitch)
	}
	if yaw, ok := data["yaw"].(float64); ok {
		payload.Yaw = float32(yaw)
	}

	// Wrap in Message
	msg := messages.NewBaseMessage(
		payload.Schema(),
		payload,
		"robotics-processor",
	)

	// Marshal and publish
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		rp.logger.Error("Error marshaling attitude message", "error", err)
		return
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(30) // ATTITUDE
	rp.mu.RUnlock()
	nc.Publish(subject, msgBytes)
}

// publishBatteryFromJSON creates and publishes a battery message from JSON
func (rp *RoboticsProcessor) publishBatteryFromJSON(data map[string]any) {
	nc := rp.GetNATSConnection()
	if nc == nil {
		return
	}

	systemID, ok := data["system_id"].(float64)
	if !ok {
		return
	}

	batteryID := uint8(0)
	if bid, ok := data["battery_id"].(float64); ok {
		batteryID = uint8(bid)
	}

	// Create battery message
	payload := payloads.NewBatteryPayload(uint8(systemID), batteryID, time.Now())

	// Set optional fields
	if remaining, ok := data["battery_remaining"].(float64); ok {
		payload.BatteryRemaining = int8(remaining)
	}
	if voltage, ok := data["voltage"].(float64); ok {
		payload.TotalVoltage = voltage
	}

	// Wrap in Message
	msg := messages.NewBaseMessage(
		payload.Schema(),
		payload,
		"robotics-processor",
	)

	// Marshal and publish
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		rp.logger.Error("Error marshaling battery message", "error", err)
		return
	}

	rp.mu.RLock()
	subject := rp.getOutputSubject(147) // BATTERY_STATUS
	rp.mu.RUnlock()
	nc.Publish(subject, msgBytes)
}

// getOutputSubject determines the NATS subject for publishing semantic messages
func (rp *RoboticsProcessor) getOutputSubject(messageID uint32) string {
	// Get configured output ports
	outputPorts := rp.OutputPorts()

	// For robotics, we typically have a single output port for all semantic messages
	// This follows the pattern of sending all messages to ObjectStore for storage
	if len(outputPorts) > 0 {
		// Use the first output port's subject (typically "storage_write" port)
		if natsPort, ok := outputPorts[0].Config.(component.NATSPort); ok {
			return natsPort.Subject
		}
	}

	// Fallback to convention if no ports configured (shouldn't happen with DefaultConfig)
	return "storage.objectstore.write"
}

// getMessageTypeName returns a human-readable name for a MAVLink message ID
func (rp *RoboticsProcessor) getMessageTypeName(messageID uint32) string {
	switch messageID {
	case 0:
		return "HEARTBEAT"
	case 33:
		return "GLOBAL_POSITION_INT"
	case 30:
		return "ATTITUDE"
	case 147:
		return "BATTERY_STATUS"
	default:
		return fmt.Sprintf("MESSAGE_%d", messageID)
	}
}

// getMessageCategory returns the message category for a given MAVLink message ID
func (rp *RoboticsProcessor) getMessageCategory(messageID uint32) string {
	switch messageID {
	case 0: // HEARTBEAT
		return "heartbeat"
	case 33: // GLOBAL_POSITION_INT
		return "position"
	case 30: // ATTITUDE
		return "attitude"
	case 147: // BATTERY_STATUS
		return "battery"
	case 24: // GPS_RAW_INT
		return "gps"
	case 1: // SYS_STATUS
		return "status"
	default:
		return fmt.Sprintf("message_%d", messageID)
	}
}

// isMessageTypeEnabled checks if a message type is enabled based on configuration
func (rp *RoboticsProcessor) isMessageTypeEnabled(messageID uint32) bool {
	rp.mu.RLock()
	config := rp.config
	rp.mu.RUnlock()

	switch messageID {
	case 0: // HEARTBEAT
		return config.ProcessHeartbeat
	case 33: // GLOBAL_POSITION_INT
		return config.ProcessPosition
	case 30: // ATTITUDE
		return config.ProcessAttitude
	case 147: // BATTERY_STATUS
		return config.ProcessBattery
	case 24: // GPS_RAW_INT
		return config.ProcessGPS
	case 1: // SYS_STATUS
		return config.ProcessStatus
	default:
		// Unknown message types are processed if any category is enabled
		return config.ProcessHeartbeat || config.ProcessPosition || config.ProcessBattery ||
			config.ProcessAttitude || config.ProcessGPS || config.ProcessStatus
	}
}

// Register registers the robotics processor component with the given registry
// Passes roboticsSchema as static metadata (Option 1 pattern - schema as metadata)
func Register(registry *component.Registry) error {
	return registry.RegisterProcessor(
		"robotics",
		CreateRoboticsProcessor,
		roboticsSchema,
		"mavlink",
		"robotics",
		"MAVLink protocol processor for autonomous vehicles",
		"1.0.0",
	)
}

// CreateRoboticsProcessor creates a robotics processor with ComponentConfig
func CreateRoboticsProcessor(rawConfig json.RawMessage, deps component.ComponentDependencies) (component.Discoverable, error) {
	// Validate required dependencies
	if deps.NATSClient == nil {
		return nil, errors.WrapInvalid(fmt.Errorf("NATS client is required"),
			"robotics-processor-factory", "create", "NATS client validation")
	}

	// Start with defaults (includes PortGenerator conventions)
	procConfig := DefaultConfig()

	// Override with user config if provided
	if len(rawConfig) > 0 {
		if err := json.Unmarshal(rawConfig, &procConfig); err != nil {
			return nil, errors.WrapInvalid(err, "robotics-processor-factory", "create", "parse config")
		}

		// Preserve default ports if not explicitly configured
		// This prevents JSON unmarshaling empty config {} from wiping out defaults
		if procConfig.Ports == nil {
			procConfig.Ports = DefaultConfig().Ports
		}
	}

	// Create processor with dependency injection
	processor, err := NewRoboticsProcessorWithMetrics(deps.NATSClient.GetConnection(), deps.MetricsRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "RoboticsProcessor", "CreateRoboticsProcessor", "create robotics processor")
	}

	// Store organization and platform for entity ID generation
	processor.organization = deps.Platform.Org
	processor.platform = deps.Platform.Platform

	// Set logger from ComponentDependencies
	processor.logger = deps.GetLoggerWithComponent("robotics-processor")

	// Apply configuration
	processor.ApplyConfig(procConfig)
	return processor, nil
}
