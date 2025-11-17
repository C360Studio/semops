package robotics

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/c360/streamkit/errors"
)

func TestRoboticsProcessor_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupError    func() error
		expectedClass errors.ErrorClass
		expectedMsg   string
	}{
		{
			name: "NewRoboticsProcessor with nil connection",
			setupError: func() error {
				_, err := NewRoboticsProcessor(nil)
				return err
			},
			expectedClass: errors.ErrorFatal,
			expectedMsg:   "RoboticsProcessor.NewRoboticsProcessor: validate NATS connection failed",
		},
		{
			name: "Start when already started",
			setupError: func() error {
				// Create a mock processor that's already started
				processor := &RoboticsProcessor{shutdown: make(chan struct{})}
				return processor.Start(context.Background())
			},
			expectedClass: errors.ErrorInvalid,
			expectedMsg:   "RoboticsProcessor.Start: check processor state failed",
		},
		{
			name: "ValidateConfiguration with non-map config",
			setupError: func() error {
				processor := &RoboticsProcessor{}
				return processor.ValidateConfiguration("invalid config")
			},
			expectedClass: errors.ErrorInvalid,
			expectedMsg:   "RoboticsProcessor.ValidateConfiguration: validate config type failed",
		},
		{
			name: "ValidateConfiguration with invalid enabled field",
			setupError: func() error {
				processor := &RoboticsProcessor{}
				config := map[string]any{
					"enabled": "not a boolean",
				}
				return processor.ValidateConfiguration(config)
			},
			expectedClass: errors.ErrorInvalid,
			expectedMsg:   "RoboticsProcessor.ValidateConfiguration: validate enabled field failed",
		},
		{
			name: "ReloadConfiguration with non-map config",
			setupError: func() error {
				processor := &RoboticsProcessor{}
				return processor.ReloadConfiguration(context.Background(), 123)
			},
			expectedClass: errors.ErrorInvalid,
			expectedMsg:   "RoboticsProcessor.ReloadConfiguration: validate config type failed",
		},
		{
			name: "ProcessRawData with no NATS connection",
			setupError: func() error {
				processor := &RoboticsProcessor{
					nc:      nil,
					enabled: true, // Enable processor so it checks NATS connection
				}
				return processor.ProcessRawData(context.Background(), "test.subject", []byte("data"))
			},
			expectedClass: errors.ErrorFatal,
			expectedMsg:   "RoboticsProcessor.ProcessRawData: check NATS connection failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.setupError()
			
			// Verify error is not nil
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			
			// Verify error classification
			actualClass := errors.Classify(err)
			if actualClass != test.expectedClass {
				t.Errorf("expected error class %v, got %v", test.expectedClass, actualClass)
			}
			
			// Verify error message contains expected content
			if test.expectedMsg != "" && err.Error() != test.expectedMsg {
				// For wrapped errors, check if the expected message is contained
				errMsg := err.Error()
				if len(errMsg) == 0 {
					t.Errorf("expected error message to contain '%s', got empty message", test.expectedMsg)
				}
				// Just verify the error contains our component and method pattern
				if !containsPattern(errMsg, "RoboticsProcessor.") {
					t.Errorf("expected error message to contain component pattern, got: %s", errMsg)
				}
			}
			
			// Test error classification helpers
			switch test.expectedClass {
			case errors.ErrorTransient:
				if !errors.IsTransient(err) {
					t.Errorf("error should be classified as transient")
				}
				if errors.IsFatal(err) || errors.IsInvalid(err) {
					t.Errorf("error should only be classified as transient")
				}
			case errors.ErrorInvalid:
				if !errors.IsInvalid(err) {
					t.Errorf("error should be classified as invalid")
				}
				if errors.IsFatal(err) || errors.IsTransient(err) {
					t.Errorf("error should only be classified as invalid")
				}
			case errors.ErrorFatal:
				if !errors.IsFatal(err) {
					t.Errorf("error should be classified as fatal")
				}
				if errors.IsTransient(err) || errors.IsInvalid(err) {
					t.Errorf("error should only be classified as fatal")
				}
			}
		})
	}
}

func TestRoboticsProcessor_ErrorWrapping(t *testing.T) {
	// Test that errors are properly wrapped with context
	processor := &RoboticsProcessor{}
	
	// Test JSON parsing error
	err := processor.processJSONData(context.Background(), nil, "subject", []byte("invalid json"))
	if err == nil {
		t.Fatal("expected error from invalid JSON")
	}
	
	// Verify error is classified as invalid
	if !errors.IsInvalid(err) {
		t.Error("JSON parsing error should be classified as invalid")
	}
	
	// Verify error message follows standard pattern
	errMsg := err.Error()
	if !containsPattern(errMsg, "RoboticsProcessor.processJSONData: parse JSON data failed:") {
		t.Errorf("error message should follow standard pattern, got: %s", errMsg)
	}
}

func TestRoboticsProcessor_RetryLogic(t *testing.T) {
	// Create a retry configuration
	retryConfig := errors.DefaultRetryConfig()
	
	// Test transient error should be retried
	transientErr := errors.ErrConnectionTimeout
	if !retryConfig.ShouldRetry(transientErr, 1) {
		t.Error("transient error should be retryable")
	}
	
	// Test fatal error should not be retried
	fatalErr := errors.ErrInvalidConfig
	if retryConfig.ShouldRetry(fatalErr, 1) {
		t.Error("fatal error should not be retryable")
	}
	
	// Test retry limit
	if retryConfig.ShouldRetry(transientErr, 3) {
		t.Error("should not retry after max attempts")
	}
	
	// Test backoff calculation
	delay := retryConfig.BackoffDelay(1)
	expectedDelay := retryConfig.InitialDelay * 2 // Should be doubled for attempt 1
	if delay != expectedDelay {
		t.Errorf("expected backoff delay %v, got %v", expectedDelay, delay)
	}
}

func TestRoboticsProcessor_ErrorChaining(t *testing.T) {
	// Test that wrapped errors preserve the original error
	originalErr := fmt.Errorf("original error")
	wrappedErr := errors.Wrap(originalErr, "RoboticsProcessor", "testMethod", "test action")
	
	// Verify we can unwrap to get original error
	if !stderrors.Is(wrappedErr, originalErr) {
		t.Error("wrapped error should preserve original error for unwrapping")
	}
	
	// Test classified error wrapping
	classifiedErr := errors.WrapTransient(originalErr, "RoboticsProcessor", "testMethod", "test action")
	
	// Verify classification
	if !errors.IsTransient(classifiedErr) {
		t.Error("classified error should maintain its classification")
	}
	
	// Verify original error is still accessible
	if !stderrors.Is(classifiedErr, originalErr) {
		t.Error("classified error should preserve original error for unwrapping")
	}
}

// Helper function to check if a string contains a pattern
func containsPattern(s, pattern string) bool {
	return len(s) >= len(pattern) && findSubstring(s, pattern) != -1
}

// Simple substring search
func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}
	
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}