package command

import (
	"strings"
	"testing"
)

func TestStatusValidationAcceptsKnownStatusesAndDefault(t *testing.T) {
	for _, status := range []string{
		"",
		StatusRequested,
		StatusAccepted,
		StatusExecuting,
		StatusCancelRequested,
		StatusCancelled,
		StatusSuperseded,
		StatusSucceeded,
		StatusFailed,
		StatusRejected,
		StatusExpired,
		StatusTimeout,
		StatusDuplicate,
		" ACCEPTED ",
	} {
		if err := validateStatus(status); err != nil {
			t.Fatalf("validate status %q: %v", status, err)
		}
	}
	if got := statusOrDefault(""); got != StatusRequested {
		t.Fatalf("default status = %q, want %q", got, StatusRequested)
	}
}

func TestStatusValidationRejectsUnknownStatus(t *testing.T) {
	err := validateStatus("approved")
	if err == nil {
		t.Fatal("expected unknown status error")
	}
	if !strings.Contains(err.Error(), "approved") {
		t.Fatalf("error = %v, want unknown status", err)
	}
}

func TestLifecycleStatusClasses(t *testing.T) {
	for _, status := range []string{StatusCancelled, StatusDuplicate, StatusExpired, StatusFailed, StatusRejected, StatusSucceeded, StatusSuperseded, StatusTimeout} {
		if !isTerminalStatus(status) {
			t.Fatalf("%q should be terminal", status)
		}
	}
	for _, status := range []string{StatusRequested, StatusAccepted, StatusExecuting, StatusCancelRequested} {
		if isTerminalStatus(status) {
			t.Fatalf("%q should not be terminal", status)
		}
	}
	if !nativeExecutionEligible(StatusAccepted) {
		t.Fatalf("%q should be native-execution eligible", StatusAccepted)
	}
	for _, status := range []string{"", StatusRequested, StatusExecuting, StatusSuperseded, StatusCancelled} {
		if nativeExecutionEligible(status) {
			t.Fatalf("%q should not be native-execution eligible", status)
		}
	}
}

func TestValidateStatusTransitionAllowsLifecycleProgression(t *testing.T) {
	for _, tt := range []struct {
		from string
		to   string
	}{
		{from: "", to: StatusRequested},
		{from: StatusRequested, to: StatusAccepted},
		{from: StatusRequested, to: StatusSuperseded},
		{from: StatusRequested, to: StatusCancelled},
		{from: StatusAccepted, to: StatusExecuting},
		{from: StatusAccepted, to: StatusCancelRequested},
		{from: StatusAccepted, to: StatusSuperseded},
		{from: StatusExecuting, to: StatusSucceeded},
		{from: StatusExecuting, to: StatusFailed},
		{from: StatusCancelRequested, to: StatusCancelled},
		{from: StatusSucceeded, to: StatusSucceeded},
	} {
		if err := ValidateStatusTransition(tt.from, tt.to); err != nil {
			t.Fatalf("transition %q -> %q: %v", tt.from, tt.to, err)
		}
	}
}

func TestValidateStatusTransitionRejectsUnsafeTransitions(t *testing.T) {
	for _, tt := range []struct {
		from string
		to   string
		want string
	}{
		{from: StatusRequested, to: StatusExecuting, want: "not allowed"},
		{from: StatusExecuting, to: StatusAccepted, want: "not allowed"},
		{from: StatusSucceeded, to: StatusAccepted, want: "terminal"},
		{from: StatusAccepted, to: "approved", want: "approved"},
	} {
		err := ValidateStatusTransition(tt.from, tt.to)
		if err == nil {
			t.Fatalf("transition %q -> %q should fail", tt.from, tt.to)
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("transition %q -> %q error = %v, want %q", tt.from, tt.to, err, tt.want)
		}
	}
}
