package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	mavlink "github.com/c360studio/semops/pkg/adapters/mavlink"
)

const (
	actionRequestAutopilotVersion = "request_autopilot_version"
	defaultRoute                  = "127.0.0.1:14540"
	defaultSourceSystem           = 255
	defaultSourceComponent        = 190
	defaultTargetSystem           = 1
	defaultTargetComponent        = 1
	defaultTimeout                = 2 * time.Second
	defaultReplyTimeout           = 2 * time.Second
	defaultHeartbeatSettle        = 250 * time.Millisecond
	replyBufferBytes              = mavlink.MaxPayloadLength + 64
)

type config struct {
	Route            string
	Action           string
	SourceSystem     int
	SourceComponent  int
	TargetSystem     int
	TargetComponent  int
	ForwardRepliesTo string
	SendHeartbeat    bool
	SimulatorOnly    bool
	DryRun           bool
	Timeout          time.Duration
	ReplyTimeout     time.Duration
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	cfg, err := defaultConfig()
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("semops-mavlink-command", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&cfg.Route, "route", cfg.Route, "simulator UDP destination, host:port or udp://host:port")
	fs.StringVar(&cfg.Action, "action", cfg.Action, "allowlisted command action")
	fs.IntVar(&cfg.SourceSystem, "source-system", cfg.SourceSystem, "MAVLink source system ID for the GCS command")
	fs.IntVar(&cfg.SourceComponent, "source-component", cfg.SourceComponent, "MAVLink source component ID for the GCS command")
	fs.IntVar(&cfg.TargetSystem, "target-system", cfg.TargetSystem, "MAVLink target system ID")
	fs.IntVar(&cfg.TargetComponent, "target-component", cfg.TargetComponent, "MAVLink target component ID")
	fs.StringVar(&cfg.ForwardRepliesTo, "forward-replies-to", cfg.ForwardRepliesTo, "optional UDP route that receives simulator reply frames observed by this helper")
	fs.BoolVar(&cfg.SendHeartbeat, "send-heartbeat-first", cfg.SendHeartbeat, "send a GCS heartbeat before the command so simulators can learn the endpoint")
	fs.BoolVar(&cfg.SimulatorOnly, "confirm-simulator-only", cfg.SimulatorOnly, "confirm this command is routed only to a simulator")
	fs.BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "print command metadata without sending the frame")
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "UDP write timeout")
	fs.DurationVar(&cfg.ReplyTimeout, "reply-timeout", cfg.ReplyTimeout, "how long to read simulator replies when forwarding is enabled")
	if err := fs.Parse(args); err != nil {
		return err
	}

	frame, summary, err := buildCommandFrame(cfg)
	if err != nil {
		return err
	}
	if cfg.DryRun {
		fmt.Fprintln(stdout, summary)
		fmt.Fprintf(stdout, "frame_hex=%s\n", hex.EncodeToString(frame))
		return nil
	}

	route, err := normalizeUDPRoute(cfg.Route)
	if err != nil {
		return err
	}
	dialer := net.Dialer{Timeout: cfg.Timeout}
	conn, err := dialer.Dial("udp", route)
	if err != nil {
		return fmt.Errorf("dial simulator UDP route %s: %w", route, err)
	}
	defer conn.Close()
	if err := conn.SetWriteDeadline(time.Now().Add(cfg.Timeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}
	if cfg.SendHeartbeat {
		heartbeat, err := buildGCSHeartbeatFrame(cfg)
		if err != nil {
			return err
		}
		if _, err := conn.Write(heartbeat); err != nil {
			return fmt.Errorf("write MAVLink GCS heartbeat: %w", err)
		}
		fmt.Fprintf(stdout, "sent_heartbeat_bytes=%d route=%s\n", len(heartbeat), route)
		time.Sleep(defaultHeartbeatSettle)
	}
	if _, err := conn.Write(frame); err != nil {
		return fmt.Errorf("write MAVLink command: %w", err)
	}
	fmt.Fprintln(stdout, summary)
	fmt.Fprintf(stdout, "sent_bytes=%d route=%s\n", len(frame), route)
	if cfg.ForwardRepliesTo != "" {
		if _, err := forwardReplies(conn, cfg.ForwardRepliesTo, cfg.ReplyTimeout, stdout); err != nil {
			return err
		}
	}
	return nil
}

func defaultConfig() (config, error) {
	timeout := defaultTimeout
	if value := os.Getenv("SEMOPS_MAVLINK_COMMAND_SEND_TIMEOUT"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return config{}, fmt.Errorf("SEMOPS_MAVLINK_COMMAND_SEND_TIMEOUT=%q is not a duration: %w", value, err)
		}
		timeout = parsed
	}
	replyTimeout := defaultReplyTimeout
	if value := os.Getenv("SEMOPS_MAVLINK_COMMAND_REPLY_TIMEOUT"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return config{}, fmt.Errorf("SEMOPS_MAVLINK_COMMAND_REPLY_TIMEOUT=%q is not a duration: %w", value, err)
		}
		replyTimeout = parsed
	}
	sourceSystem, err := envInt("SEMOPS_MAVLINK_COMMAND_SOURCE_SYSTEM", defaultSourceSystem)
	if err != nil {
		return config{}, err
	}
	sourceComponent, err := envInt("SEMOPS_MAVLINK_COMMAND_SOURCE_COMPONENT", defaultSourceComponent)
	if err != nil {
		return config{}, err
	}
	targetSystem, err := envInt("SEMOPS_MAVLINK_COMMAND_TARGET_SYSTEM", defaultTargetSystem)
	if err != nil {
		return config{}, err
	}
	targetComponent, err := envInt("SEMOPS_MAVLINK_COMMAND_TARGET_COMPONENT", defaultTargetComponent)
	if err != nil {
		return config{}, err
	}
	simulatorOnly, err := envBool("SEMOPS_MAVLINK_COMMAND_SIMULATOR_ONLY_CONFIRMED", false)
	if err != nil {
		return config{}, err
	}
	sendHeartbeat, err := envBool("SEMOPS_MAVLINK_COMMAND_SEND_HEARTBEAT_FIRST", false)
	if err != nil {
		return config{}, err
	}
	return config{
		Route:            firstNonEmpty(os.Getenv("SEMOPS_MAVLINK_COMMAND_UDP_ROUTE"), defaultRoute),
		Action:           firstNonEmpty(os.Getenv("SEMOPS_MAVLINK_COMMAND_ACTION"), actionRequestAutopilotVersion),
		SourceSystem:     sourceSystem,
		SourceComponent:  sourceComponent,
		TargetSystem:     targetSystem,
		TargetComponent:  targetComponent,
		ForwardRepliesTo: os.Getenv("SEMOPS_MAVLINK_COMMAND_FORWARD_REPLIES_TO"),
		SendHeartbeat:    sendHeartbeat,
		SimulatorOnly:    simulatorOnly,
		Timeout:          timeout,
		ReplyTimeout:     replyTimeout,
	}, nil
}

func forwardReplies(conn net.Conn, forwardRoute string, timeout time.Duration, stdout io.Writer) (int, error) {
	if timeout <= 0 {
		return 0, errors.New("reply timeout must be greater than zero when forwarding replies")
	}
	route, err := normalizeUDPRoute(forwardRoute)
	if err != nil {
		return 0, fmt.Errorf("normalize reply forward route: %w", err)
	}
	forwardConn, err := net.DialTimeout("udp", route, timeout)
	if err != nil {
		return 0, fmt.Errorf("dial reply forward route %s: %w", route, err)
	}
	defer forwardConn.Close()

	deadline := time.Now().Add(timeout)
	buffer := make([]byte, replyBufferBytes)
	forwarded := 0
	for {
		if err := conn.SetReadDeadline(deadline); err != nil {
			return forwarded, fmt.Errorf("set reply read deadline: %w", err)
		}
		n, err := conn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			return forwarded, fmt.Errorf("read simulator reply: %w", err)
		}
		if n == 0 {
			continue
		}
		if _, err := forwardConn.Write(buffer[:n]); err != nil {
			return forwarded, fmt.Errorf("forward simulator reply to %s: %w", route, err)
		}
		forwarded++
		fmt.Fprintf(stdout, "forwarded_reply_bytes=%d route=%s\n", n, route)
	}
	fmt.Fprintf(stdout, "forwarded_replies=%d route=%s\n", forwarded, route)
	return forwarded, nil
}

func buildGCSHeartbeatFrame(cfg config) ([]byte, error) {
	sourceSystem, err := uint8ID("source-system", cfg.SourceSystem, false)
	if err != nil {
		return nil, err
	}
	sourceComponent, err := uint8ID("source-component", cfg.SourceComponent, false)
	if err != nil {
		return nil, err
	}
	generator := mavlink.NewGenerator(sourceSystem, sourceComponent)
	return generator.GenerateHeartbeat(mavlink.HeartbeatMessage{
		VehicleType:    mavlink.TypeGCS,
		Autopilot:      mavlink.AutopilotInvalid,
		SystemStatus:   mavlink.StateActive,
		MavlinkVersion: 3,
	})
}

func buildCommandFrame(cfg config) ([]byte, string, error) {
	if !cfg.SimulatorOnly {
		return nil, "", errors.New("simulator-only confirmation is required; pass -confirm-simulator-only or set SEMOPS_MAVLINK_COMMAND_SIMULATOR_ONLY_CONFIRMED=true")
	}
	sourceSystem, err := uint8ID("source-system", cfg.SourceSystem, false)
	if err != nil {
		return nil, "", err
	}
	sourceComponent, err := uint8ID("source-component", cfg.SourceComponent, false)
	if err != nil {
		return nil, "", err
	}
	targetSystem, err := uint8ID("target-system", cfg.TargetSystem, false)
	if err != nil {
		return nil, "", err
	}
	targetComponent, err := uint8ID("target-component", cfg.TargetComponent, true)
	if err != nil {
		return nil, "", err
	}

	command, params, err := commandForAction(cfg.Action)
	if err != nil {
		return nil, "", err
	}
	generator := mavlink.NewGenerator(sourceSystem, sourceComponent)
	frame, err := generator.GenerateCommandLong(mavlink.CommandLongMessage{
		Command:           command,
		TargetSystemID:    targetSystem,
		TargetComponentID: targetComponent,
		Params:            params,
	})
	if err != nil {
		return nil, "", fmt.Errorf("generate command frame: %w", err)
	}

	summary := fmt.Sprintf(
		"action=%s command=%d request_message=%d source=%d/%d target=%d/%d expected_ack_task_suffix=system-%d-command-%d-target-%d-%d",
		cfg.Action,
		command,
		mavlink.MessageIDAutopilotVersion,
		sourceSystem,
		sourceComponent,
		targetSystem,
		targetComponent,
		targetSystem,
		command,
		sourceSystem,
		sourceComponent,
	)
	return frame, summary, nil
}

func commandForAction(action string) (uint16, [7]float32, error) {
	switch strings.TrimSpace(action) {
	case actionRequestAutopilotVersion:
		return mavlink.CommandRequestMessage, [7]float32{float32(mavlink.MessageIDAutopilotVersion)}, nil
	default:
		return 0, [7]float32{}, fmt.Errorf("unsupported action %q; MVP allowlist: %s", action, actionRequestAutopilotVersion)
	}
}

func normalizeUDPRoute(route string) (string, error) {
	route = strings.TrimSpace(route)
	route = strings.TrimPrefix(route, "udp://")
	if route == "" {
		return "", errors.New("simulator UDP route is required")
	}
	if strings.HasPrefix(route, ":") {
		return "", fmt.Errorf("simulator UDP route %q is listen-style; provide a destination host:port such as %s", route, defaultRoute)
	}
	if _, _, err := net.SplitHostPort(route); err != nil {
		return "", fmt.Errorf("simulator UDP route %q must be host:port or udp://host:port: %w", route, err)
	}
	return route, nil
}

func uint8ID(name string, value int, allowZero bool) (uint8, error) {
	min := 1
	if allowZero {
		min = 0
	}
	if value < min || value > 255 {
		return 0, fmt.Errorf("%s must be between %d and 255, got %d", name, min, value)
	}
	return uint8(value), nil
}

func envInt(name string, defaultValue int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not an integer: %w", name, value, err)
	}
	return parsed, nil
}

func envBool(name string, defaultValue bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue, nil
	}
	switch value {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes", "on", "ON", "On":
		return true, nil
	case "0", "false", "FALSE", "False", "no", "NO", "No", "off", "OFF", "Off":
		return false, nil
	default:
		return false, fmt.Errorf("%s=%q is not a boolean", name, value)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
