package sitl

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockUDPConn is a mock UDP connection for testing
type MockUDPConn struct {
	readData   []byte
	writeData  []byte
	closed     bool
	readError  error
	writeError error
}

func NewMockUDPConn() *MockUDPConn {
	return &MockUDPConn{}
}

func (m *MockUDPConn) Read(b []byte) (n int, err error) {
	if m.readError != nil {
		return 0, m.readError
	}
	if len(m.readData) == 0 {
		// Simulate timeout
		time.Sleep(50 * time.Millisecond)
		return 0, &net.OpError{Op: "read", Err: &timeoutError{}}
	}
	n = copy(b, m.readData)
	m.readData = m.readData[n:]
	return n, nil
}

func (m *MockUDPConn) Write(b []byte) (n int, err error) {
	if m.writeError != nil {
		return 0, m.writeError
	}
	m.writeData = append(m.writeData, b...)
	return len(b), nil
}

func (m *MockUDPConn) Close() error {
	m.closed = true
	return nil
}

func (m *MockUDPConn) LocalAddr() net.Addr                { return nil }
func (m *MockUDPConn) RemoteAddr() net.Addr               { return nil }
func (m *MockUDPConn) SetDeadline(_ time.Time) error      { return nil }
func (m *MockUDPConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *MockUDPConn) SetWriteDeadline(_ time.Time) error { return nil }

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func TestNewController(t *testing.T) {
	tests := []struct {
		name   string
		config ControllerConfig
	}{
		{
			name:   "default config",
			config: DefaultConfig(),
		},
		{
			name: "custom config",
			config: ControllerConfig{
				Address:           "127.0.0.1:14551",
				SystemID:          100,
				ComponentID:       2,
				TargetSystemID:    2,
				TargetComponentID: 2,
				CommandTimeout:    3 * time.Second,
				HeartbeatTimeout:  15 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip this test if we can't create a real UDP connection
			// This test would need a mock dialer or similar for true unit testing
			t.Skip("Skipping connection test - requires mock dialer implementation")
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.Equal(t, "localhost:5760", config.Address)
	assert.Equal(t, uint8(255), config.SystemID)
	assert.Equal(t, uint8(1), config.ComponentID)
	assert.Equal(t, uint8(1), config.TargetSystemID)
	assert.Equal(t, uint8(1), config.TargetComponentID)
	assert.Equal(t, 5*time.Second, config.CommandTimeout)
	assert.Equal(t, 10*time.Second, config.HeartbeatTimeout)
}

func TestDroneState(t *testing.T) {
	state := &DroneState{
		Connected:      true,
		Armed:          false,
		FlightMode:     "STABILIZE",
		Latitude:       37.7749,
		Longitude:      -122.4194,
		Altitude:       100.5,
		RelativeAlt:    50.2,
		BatteryPercent: 85,
	}
	
	assert.True(t, state.Connected)
	assert.False(t, state.Armed)
	assert.Equal(t, "STABILIZE", state.FlightMode)
	assert.Equal(t, 37.7749, state.Latitude)
	assert.Equal(t, -122.4194, state.Longitude)
	assert.Equal(t, 100.5, state.Altitude)
	assert.Equal(t, 50.2, state.RelativeAlt)
	assert.Equal(t, int8(85), state.BatteryPercent)
}

func TestCommandResult(t *testing.T) {
	result := CommandResult{
		Command: 400, // MAV_CMD_COMPONENT_ARM_DISARM
		Result:  0,   // MAV_RESULT_ACCEPTED
		Success: true,
		Message: "Command accepted",
	}
	
	assert.Equal(t, uint16(400), result.Command)
	assert.Equal(t, uint8(0), result.Result)
	assert.True(t, result.Success)
	assert.Equal(t, "Command accepted", result.Message)
}

func TestStringToCustomMode(t *testing.T) {
	ctrl := &Controller{} // Empty controller for method access
	
	tests := []struct {
		mode string
		want uint32
		err  bool
	}{
		{"STABILIZE", 0, false},
		{"ACRO", 1, false},
		{"ALT_HOLD", 2, false},
		{"AUTO", 3, false},
		{"GUIDED", 4, false},
		{"LOITER", 5, false},
		{"RTL", 6, false},
		{"LAND", 9, false},
		{"INVALID_MODE", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got, err := ctrl.stringToCustomMode(tt.mode)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCustomModeToString(t *testing.T) {
	ctrl := &Controller{} // Empty controller for method access
	
	tests := []struct {
		mode uint32
		want string
	}{
		{0, "STABILIZE"},
		{1, "ACRO"},
		{2, "ALT_HOLD"},
		{3, "AUTO"},
		{4, "GUIDED"},
		{5, "LOITER"},
		{6, "RTL"},
		{9, "LAND"},
		{999, "UNKNOWN(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := ctrl.customModeToString(tt.mode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMavResultToString(t *testing.T) {
	ctrl := &Controller{} // Empty controller for method access
	
	tests := []struct {
		result uint8
		want   string
	}{
		{0, "Command accepted"},
		{1, "Command temporarily rejected"},
		{2, "Command denied"},
		{3, "Command unsupported"},
		{4, "Command failed"},
		{5, "Command in progress"},
		{99, "Unknown result (99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := ctrl.mavResultToString(tt.result)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEncodeCommandLong(t *testing.T) {
	ctrl := &Controller{
		targetSysID:  1,
		targetCompID: 1,
	}
	
	params := [7]float32{1, 0, 0, 0, 0, 0, 0}
	payload := ctrl.encodeCommandLong(400, params) // ARM command
	
	require.Equal(t, 33, len(payload))
	
	// Check command ID at offset 28
	command := uint16(payload[28]) | (uint16(payload[29]) << 8)
	assert.Equal(t, uint16(400), command)
	
	// Check target system at offset 30
	assert.Equal(t, uint8(1), payload[30])
	
	// Check target component at offset 31  
	assert.Equal(t, uint8(1), payload[31])
	
	// Check confirmation at offset 32
	assert.Equal(t, uint8(0), payload[32])
}

func TestGetSystemStatusName(t *testing.T) {
	ctrl := &Controller{} // Empty controller for method access
	
	tests := []struct {
		status uint8
		want   string
	}{
		{0, "Uninitialized"},
		{1, "Booting"},
		{2, "Calibrating"},
		{3, "Standby"},
		{4, "Active"},
		{5, "Critical"},
		{6, "Emergency"},
		{7, "Power Off"},
		{8, "Flight Termination"},
		{99, "Unknown (99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := ctrl.getSystemStatusName(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWaypoint(t *testing.T) {
	waypoint := Waypoint{
		Latitude:  37.7749,
		Longitude: -122.4194,
		Altitude:  100.0,
	}
	
	assert.Equal(t, 37.7749, waypoint.Latitude)
	assert.Equal(t, -122.4194, waypoint.Longitude)
	assert.Equal(t, 100.0, waypoint.Altitude)
}

func TestFlightPlan(t *testing.T) {
	waypoints := []Waypoint{
		{37.7749, -122.4194, 100.0},
		{37.7750, -122.4195, 100.0},
	}
	
	plan := FlightPlan{
		Waypoints: waypoints,
		HoldTime:  5 * time.Second,
		Tolerance: 5.0,
		Timeout:   60 * time.Second,
	}
	
	assert.Equal(t, 2, len(plan.Waypoints))
	assert.Equal(t, 5*time.Second, plan.HoldTime)
	assert.Equal(t, 5.0, plan.Tolerance)
	assert.Equal(t, 60*time.Second, plan.Timeout)
}

// Integration test helpers (require running SITL)

func TestBasicFlightIntegration(t *testing.T) {
	if !isSITLRunning() {
		t.Skip("SITL not running")
	}
	
	ctx := context.Background()
	config := DefaultConfig()
	
	ctrl, err := NewController(ctx, config)
	require.NoError(t, err)
	defer ctrl.Close()
	
	// Wait for connection
	time.Sleep(2 * time.Second)
	require.True(t, ctrl.IsConnected(), "Should be connected to SITL")
	
	// Test basic flight at low altitude for safety
	err = BasicFlight(ctx, ctrl, 5.0)
	assert.NoError(t, err)
}

// isSITLRunning checks if SITL is available for integration tests
func isSITLRunning() bool {
	conn, err := net.DialTimeout("tcp", "localhost:5760", 1*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close() // Ignore close errors in test
	return true
}

// Benchmark tests

func BenchmarkEncodeCommandLong(b *testing.B) {
	ctrl := &Controller{
		targetSysID:  1,
		targetCompID: 1,
	}
	params := [7]float32{1, 0, 0, 0, 0, 0, 0}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctrl.encodeCommandLong(400, params)
	}
}

func BenchmarkCalculateCRC(b *testing.B) {
	ctrl := &Controller{}
	data := make([]byte, 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctrl.calculateCRC(data, 76)
	}
}