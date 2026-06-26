package main

import (
	"bytes"
	"strings"
	"testing"

	mavlink "github.com/c360studio/semops/pkg/adapters/mavlink"
)

func TestRunDryRunBuildsReadSideMVPCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"-confirm-simulator-only",
		"-dry-run",
		"-route", "udp://127.0.0.1:14540",
		"-target-system", "1",
		"-target-component", "1",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run dry-run: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	for _, want := range []string{
		"action=request_autopilot_version",
		"command=512",
		"request_message=148",
		"source=255/190",
		"target=1/1",
		"expected_ack_task_suffix=system-1-command-512-target-255-190",
		"frame_hex=",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, output)
		}
	}
}

func TestBuildCommandFrameRejectsWithoutSimulatorConfirmation(t *testing.T) {
	_, _, err := buildCommandFrame(config{Action: actionRequestAutopilotVersion})
	if err == nil {
		t.Fatal("expected simulator confirmation error")
	}
	if !strings.Contains(err.Error(), "simulator-only confirmation is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestCommandForActionOnlyAllowsReadSideMVPCommand(t *testing.T) {
	command, params, err := commandForAction(actionRequestAutopilotVersion)
	if err != nil {
		t.Fatalf("command for action: %v", err)
	}
	if command != mavlink.CommandRequestMessage {
		t.Fatalf("command = %d, want %d", command, mavlink.CommandRequestMessage)
	}
	if params[0] != float32(mavlink.MessageIDAutopilotVersion) {
		t.Fatalf("param1 = %f, want %d", params[0], mavlink.MessageIDAutopilotVersion)
	}

	if _, _, err := commandForAction("arm"); err == nil {
		t.Fatal("expected unsupported action error")
	}
}

func TestNormalizeUDPRouteRejectsListenStyleRoute(t *testing.T) {
	if _, err := normalizeUDPRoute("udp://:14540"); err == nil {
		t.Fatal("expected listen-style route error")
	}

	got, err := normalizeUDPRoute("udp://127.0.0.1:14540")
	if err != nil {
		t.Fatalf("normalize route: %v", err)
	}
	if got != "127.0.0.1:14540" {
		t.Fatalf("route = %q", got)
	}
}
