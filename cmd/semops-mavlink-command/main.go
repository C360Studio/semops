package main

import (
	"context"
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

	mavcomponent "github.com/c360studio/semops/internal/components/mavlink"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
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
	defaultLearnRouteTimeout      = 5 * time.Second
	defaultCommandAttempts        = 1
	defaultCommandRetryInterval   = 500 * time.Millisecond
	defaultHeartbeatSettle        = 250 * time.Millisecond
	replyBufferBytes              = mavcodec.MaxPayloadLength + 64
)

type config struct {
	Route             string
	Action            string
	SourceSystem      int
	SourceComponent   int
	TargetSystem      int
	TargetComponent   int
	ForwardRepliesTo  string
	LearnRouteNATSURL string
	LearnRouteSubject string
	SendHeartbeat     bool
	SimulatorOnly     bool
	DryRun            bool
	CommandAttempts   int
	Timeout           time.Duration
	ReplyTimeout      time.Duration
	LearnRouteTimeout time.Duration
	RetryInterval     time.Duration
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
	fs.StringVar(&cfg.LearnRouteNATSURL, "learn-route-nats-url", cfg.LearnRouteNATSURL, "optional NATS URL for learning the simulator UDP route from raw MAVLink telemetry")
	fs.StringVar(&cfg.LearnRouteSubject, "learn-route-subject", cfg.LearnRouteSubject, "NATS subject carrying raw MAVLink frames used for route learning")
	fs.BoolVar(&cfg.SendHeartbeat, "send-heartbeat-first", cfg.SendHeartbeat, "send a GCS heartbeat before the command so simulators can learn the endpoint")
	fs.BoolVar(&cfg.SimulatorOnly, "confirm-simulator-only", cfg.SimulatorOnly, "confirm this command is routed only to a simulator")
	fs.BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "print command metadata without sending the frame")
	fs.IntVar(&cfg.CommandAttempts, "attempts", cfg.CommandAttempts, "number of bounded command send attempts; retries increment COMMAND_LONG confirmation")
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "UDP write timeout")
	fs.DurationVar(&cfg.ReplyTimeout, "reply-timeout", cfg.ReplyTimeout, "how long to read simulator replies when forwarding is enabled")
	fs.DurationVar(&cfg.LearnRouteTimeout, "learn-route-timeout", cfg.LearnRouteTimeout, "how long to wait for raw MAVLink telemetry when learning the command route")
	fs.DurationVar(&cfg.RetryInterval, "retry-interval", cfg.RetryInterval, "reply wait between command retry attempts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	builder, err := newCommandFrameBuilder(cfg)
	if err != nil {
		return err
	}
	if cfg.DryRun {
		frame, summary, err := builder.CommandFrame(0)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, summary)
		fmt.Fprintf(stdout, "frame_hex=%s\n", hex.EncodeToString(frame))
		return nil
	}

	route := cfg.Route
	if cfg.LearnRouteNATSURL != "" {
		learned, err := learnRouteFromNATS(cfg)
		if err != nil {
			return err
		}
		route = learned
		fmt.Fprintf(stdout, "learned_route=%s nats_url=%s subject=%s target_system=%d\n",
			route, cfg.LearnRouteNATSURL, cfg.LearnRouteSubject, cfg.TargetSystem)
	}
	route, err = normalizeUDPRoute(route)
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
	return runCommandSession(conn, cfg, builder, route, stdout)
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
	learnRouteTimeout := defaultLearnRouteTimeout
	if value := os.Getenv("SEMOPS_MAVLINK_COMMAND_LEARN_ROUTE_TIMEOUT"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return config{}, fmt.Errorf("SEMOPS_MAVLINK_COMMAND_LEARN_ROUTE_TIMEOUT=%q is not a duration: %w", value, err)
		}
		learnRouteTimeout = parsed
	}
	retryInterval := defaultCommandRetryInterval
	if value := os.Getenv("SEMOPS_MAVLINK_COMMAND_RETRY_INTERVAL"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return config{}, fmt.Errorf("SEMOPS_MAVLINK_COMMAND_RETRY_INTERVAL=%q is not a duration: %w", value, err)
		}
		retryInterval = parsed
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
	commandAttempts, err := envInt("SEMOPS_MAVLINK_COMMAND_ATTEMPTS", defaultCommandAttempts)
	if err != nil {
		return config{}, err
	}
	return config{
		Route:             firstNonEmpty(os.Getenv("SEMOPS_MAVLINK_COMMAND_UDP_ROUTE"), defaultRoute),
		Action:            firstNonEmpty(os.Getenv("SEMOPS_MAVLINK_COMMAND_ACTION"), actionRequestAutopilotVersion),
		SourceSystem:      sourceSystem,
		SourceComponent:   sourceComponent,
		TargetSystem:      targetSystem,
		TargetComponent:   targetComponent,
		ForwardRepliesTo:  os.Getenv("SEMOPS_MAVLINK_COMMAND_FORWARD_REPLIES_TO"),
		LearnRouteNATSURL: os.Getenv("SEMOPS_MAVLINK_COMMAND_LEARN_ROUTE_NATS_URL"),
		LearnRouteSubject: firstNonEmpty(os.Getenv("SEMOPS_MAVLINK_COMMAND_LEARN_ROUTE_SUBJECT"), mavcomponent.DefaultRawSubject),
		SendHeartbeat:     sendHeartbeat,
		SimulatorOnly:     simulatorOnly,
		CommandAttempts:   commandAttempts,
		Timeout:           timeout,
		ReplyTimeout:      replyTimeout,
		LearnRouteTimeout: learnRouteTimeout,
		RetryInterval:     retryInterval,
	}, nil
}

func learnRouteFromNATS(cfg config) (string, error) {
	if cfg.LearnRouteTimeout <= 0 {
		return "", errors.New("learn route timeout must be greater than zero")
	}
	if strings.TrimSpace(cfg.LearnRouteSubject) == "" {
		return "", errors.New("learn route subject is required")
	}
	targetSystem, err := uint8ID("target-system", cfg.TargetSystem, false)
	if err != nil {
		return "", err
	}
	registry := payloadregistry.New()
	if err := mavcomponent.RegisterPayloads(registry); err != nil {
		return "", fmt.Errorf("register MAVLink payloads for route learning: %w", err)
	}
	decoder := message.NewDecoder(registry)
	parser := mavcodec.NewParser()

	nc, err := nats.Connect(
		cfg.LearnRouteNATSURL,
		nats.Name("semops-mavlink-command-route-learner"),
		nats.Timeout(cfg.LearnRouteTimeout),
	)
	if err != nil {
		return "", fmt.Errorf("connect NATS for MAVLink route learning: %w", err)
	}
	defer nc.Close()

	sub, err := nc.SubscribeSync(cfg.LearnRouteSubject)
	if err != nil {
		return "", fmt.Errorf("subscribe MAVLink raw subject %s: %w", cfg.LearnRouteSubject, err)
	}
	defer sub.Unsubscribe()
	if err := nc.FlushTimeout(cfg.LearnRouteTimeout); err != nil {
		return "", fmt.Errorf("flush MAVLink route learning subscription: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.LearnRouteTimeout)
	defer cancel()
	var lastErr error
	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				if lastErr != nil {
					return "", fmt.Errorf("learn MAVLink simulator route timed out after %s: %w", cfg.LearnRouteTimeout, lastErr)
				}
				return "", fmt.Errorf("learn MAVLink simulator route timed out after %s waiting for target system %d on %s",
					cfg.LearnRouteTimeout, targetSystem, cfg.LearnRouteSubject)
			}
			return "", fmt.Errorf("receive MAVLink raw frame for route learning: %w", err)
		}
		route, ok, err := routeFromRawFrame(msg.Data, decoder, parser, targetSystem)
		if err != nil {
			lastErr = err
			continue
		}
		if ok {
			return route, nil
		}
	}
}

func routeFromRawFrame(
	data []byte,
	decoder *message.Decoder,
	parser *mavcodec.Parser,
	targetSystem uint8,
) (string, bool, error) {
	if decoder == nil {
		return "", false, errors.New("route learning decoder is nil")
	}
	if parser == nil {
		parser = mavcodec.NewParser()
	}
	envelope, err := decoder.Decode(data)
	if err != nil {
		return "", false, fmt.Errorf("decode raw MAVLink route-learning payload: %w", err)
	}
	payload, ok := envelope.Payload().(*mavcomponent.RawFramePayload)
	if !ok {
		return "", false, fmt.Errorf("route-learning payload = %T, want *mavlink.RawFramePayload", envelope.Payload())
	}
	if strings.TrimSpace(payload.RemoteAddr) == "" {
		return "", false, nil
	}
	packets, err := parser.Parse(payload.Frame)
	if err != nil {
		return "", false, fmt.Errorf("parse MAVLink route-learning frame: %w", err)
	}
	if len(packets) != 1 {
		return "", false, nil
	}
	if packets[0].SystemID != targetSystem {
		return "", false, nil
	}
	return payload.RemoteAddr, true, nil
}

type commandFrameBuilder struct {
	generator       *mavcodec.Generator
	action          string
	command         uint16
	params          [7]float32
	sourceSystem    uint8
	sourceComponent uint8
	targetSystem    uint8
	targetComponent uint8
}

func newCommandFrameBuilder(cfg config) (*commandFrameBuilder, error) {
	if !cfg.SimulatorOnly {
		return nil, errors.New("simulator-only confirmation is required; pass -confirm-simulator-only or set SEMOPS_MAVLINK_COMMAND_SIMULATOR_ONLY_CONFIRMED=true")
	}
	if cfg.CommandAttempts < 1 {
		return nil, fmt.Errorf("attempts must be at least 1, got %d", cfg.CommandAttempts)
	}
	if cfg.CommandAttempts > 256 {
		return nil, fmt.Errorf("attempts must be at most 256, got %d", cfg.CommandAttempts)
	}
	if cfg.CommandAttempts > 1 && cfg.RetryInterval <= 0 {
		return nil, errors.New("retry interval must be greater than zero when attempts is greater than 1")
	}
	sourceSystem, err := uint8ID("source-system", cfg.SourceSystem, false)
	if err != nil {
		return nil, err
	}
	sourceComponent, err := uint8ID("source-component", cfg.SourceComponent, false)
	if err != nil {
		return nil, err
	}
	targetSystem, err := uint8ID("target-system", cfg.TargetSystem, false)
	if err != nil {
		return nil, err
	}
	targetComponent, err := uint8ID("target-component", cfg.TargetComponent, true)
	if err != nil {
		return nil, err
	}
	command, params, err := commandForAction(cfg.Action)
	if err != nil {
		return nil, err
	}
	return &commandFrameBuilder{
		generator:       mavcodec.NewGenerator(sourceSystem, sourceComponent),
		action:          cfg.Action,
		command:         command,
		params:          params,
		sourceSystem:    sourceSystem,
		sourceComponent: sourceComponent,
		targetSystem:    targetSystem,
		targetComponent: targetComponent,
	}, nil
}

func (b *commandFrameBuilder) GCSHeartbeatFrame() ([]byte, error) {
	return b.generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{
		VehicleType:    mavcodec.TypeGCS,
		Autopilot:      mavcodec.AutopilotInvalid,
		SystemStatus:   mavcodec.StateActive,
		MavlinkVersion: 3,
	})
}

func (b *commandFrameBuilder) CommandFrame(confirmation uint8) ([]byte, string, error) {
	frame, err := b.generator.GenerateCommandLong(mavcodec.CommandLongMessage{
		Command:           b.command,
		TargetSystemID:    b.targetSystem,
		TargetComponentID: b.targetComponent,
		Confirmation:      confirmation,
		Params:            b.params,
	})
	if err != nil {
		return nil, "", fmt.Errorf("generate command frame: %w", err)
	}
	summary := fmt.Sprintf(
		"action=%s command=%d request_message=%d source=%d/%d target=%d/%d confirmation=%d expected_ack_task_suffix=system-%d-command-%d-target-%d-%d",
		b.action,
		b.command,
		mavcodec.MessageIDAutopilotVersion,
		b.sourceSystem,
		b.sourceComponent,
		b.targetSystem,
		b.targetComponent,
		confirmation,
		b.targetSystem,
		b.command,
		b.sourceSystem,
		b.sourceComponent,
	)
	return frame, summary, nil
}

type commandReplyStats struct {
	ForwardedReplies        int
	DirectCommandACKs       int
	DirectAutopilotVersions int
	LastACKResult           string
}

func (s *commandReplyStats) Add(other commandReplyStats) {
	s.ForwardedReplies += other.ForwardedReplies
	s.DirectCommandACKs += other.DirectCommandACKs
	s.DirectAutopilotVersions += other.DirectAutopilotVersions
	if other.LastACKResult != "" {
		s.LastACKResult = other.LastACKResult
	}
}

func runCommandSession(
	conn net.Conn,
	cfg config,
	builder *commandFrameBuilder,
	route string,
	stdout io.Writer,
) error {
	total := commandReplyStats{}
	attemptsUsed := 0
	for attempt := 1; attempt <= cfg.CommandAttempts; attempt++ {
		attemptsUsed = attempt
		if err := conn.SetWriteDeadline(time.Now().Add(cfg.Timeout)); err != nil {
			return fmt.Errorf("set write deadline: %w", err)
		}
		if cfg.SendHeartbeat {
			heartbeat, err := builder.GCSHeartbeatFrame()
			if err != nil {
				return err
			}
			if _, err := conn.Write(heartbeat); err != nil {
				return fmt.Errorf("write MAVLink GCS heartbeat: %w", err)
			}
			fmt.Fprintf(stdout, "sent_heartbeat_bytes=%d route=%s attempt=%d\n", len(heartbeat), route, attempt)
			time.Sleep(defaultHeartbeatSettle)
		}

		confirmation := uint8(attempt - 1)
		frame, summary, err := builder.CommandFrame(confirmation)
		if err != nil {
			return err
		}
		if _, err := conn.Write(frame); err != nil {
			return fmt.Errorf("write MAVLink command attempt %d: %w", attempt, err)
		}
		fmt.Fprintln(stdout, summary)
		fmt.Fprintf(stdout, "sent_bytes=%d route=%s attempt=%d\n", len(frame), route, attempt)

		shouldReadReplies := cfg.ForwardRepliesTo != "" || cfg.CommandAttempts > 1
		if !shouldReadReplies {
			continue
		}
		window := cfg.ReplyTimeout
		if attempt < cfg.CommandAttempts {
			window = cfg.RetryInterval
		}
		stats, err := readSimulatorReplies(
			conn,
			cfg.ForwardRepliesTo,
			window,
			builder.command,
			stdout,
		)
		if err != nil {
			return err
		}
		total.Add(stats)
		if stats.DirectCommandACKs > 0 {
			break
		}
		if attempt < cfg.CommandAttempts {
			fmt.Fprintf(stdout, "retrying_without_direct_command_ack attempt=%d next_attempt=%d retry_interval=%s\n",
				attempt, attempt+1, cfg.RetryInterval)
		}
	}
	fmt.Fprintf(stdout, "command_attempts=%d forwarded_replies=%d direct_command_acks=%d direct_autopilot_version_frames=%d",
		attemptsUsed,
		total.ForwardedReplies,
		total.DirectCommandACKs,
		total.DirectAutopilotVersions,
	)
	if total.LastACKResult != "" {
		fmt.Fprintf(stdout, " last_ack_result=%s", total.LastACKResult)
	}
	fmt.Fprintln(stdout)
	return nil
}

func readSimulatorReplies(
	conn net.Conn,
	forwardRoute string,
	timeout time.Duration,
	command uint16,
	stdout io.Writer,
) (commandReplyStats, error) {
	if timeout <= 0 {
		return commandReplyStats{}, errors.New("reply timeout must be greater than zero when reading simulator replies")
	}
	var forwardConn net.Conn
	var route string
	if forwardRoute != "" {
		var err error
		route, err = normalizeUDPRoute(forwardRoute)
		if err != nil {
			return commandReplyStats{}, fmt.Errorf("normalize reply forward route: %w", err)
		}
		forwardConn, err = net.DialTimeout("udp", route, timeout)
		if err != nil {
			return commandReplyStats{}, fmt.Errorf("dial reply forward route %s: %w", route, err)
		}
		defer forwardConn.Close()
	}

	deadline := time.Now().Add(timeout)
	buffer := make([]byte, replyBufferBytes)
	parser := mavcodec.NewParser()
	stats := commandReplyStats{}
	for {
		if err := conn.SetReadDeadline(deadline); err != nil {
			return stats, fmt.Errorf("set reply read deadline: %w", err)
		}
		n, err := conn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			return stats, fmt.Errorf("read simulator reply: %w", err)
		}
		if n == 0 {
			continue
		}
		if forwardConn != nil {
			if _, err := forwardConn.Write(buffer[:n]); err != nil {
				return stats, fmt.Errorf("forward simulator reply to %s: %w", route, err)
			}
			stats.ForwardedReplies++
			fmt.Fprintf(stdout, "forwarded_reply_bytes=%d route=%s\n", n, route)
		}
		packets, err := parser.Parse(buffer[:n])
		if err != nil {
			return stats, fmt.Errorf("parse simulator reply: %w", err)
		}
		stats.ObservePackets(packets, command, stdout)
	}
	fmt.Fprintf(stdout, "reply_window=%s forwarded_replies=%d direct_command_acks=%d direct_autopilot_version_frames=%d\n",
		timeout, stats.ForwardedReplies, stats.DirectCommandACKs, stats.DirectAutopilotVersions)
	return stats, nil
}

func (s *commandReplyStats) ObservePackets(packets []*mavcodec.Packet, command uint16, stdout io.Writer) {
	for _, packet := range packets {
		switch packet.MessageID {
		case mavcodec.MessageIDCommandAck:
			ackedCommand, ok := packetField[uint16](packet, "command")
			if ok && ackedCommand != command {
				continue
			}
			resultValue, resultOK := packetField[uint8](packet, "result")
			result := "unknown"
			if resultOK {
				result = mavcodec.MAVResultString(resultValue)
			}
			s.DirectCommandACKs++
			s.LastACKResult = result
			fmt.Fprintf(stdout, "observed_command_ack command=%d result=%s source=%d/%d target=%s/%s\n",
				command,
				result,
				packet.SystemID,
				packet.ComponentID,
				packetFieldString[uint8](packet, "target_system"),
				packetFieldString[uint8](packet, "target_component"),
			)
		case mavcodec.MessageIDAutopilotVersion:
			s.DirectAutopilotVersions++
			fmt.Fprintf(stdout, "observed_autopilot_version source=%d/%d\n", packet.SystemID, packet.ComponentID)
		}
	}
}

func packetField[T any](packet *mavcodec.Packet, name string) (T, bool) {
	var zero T
	if packet == nil || packet.ParsedFields == nil {
		return zero, false
	}
	value, ok := packet.ParsedFields[name].(T)
	return value, ok
}

func packetFieldString[T any](packet *mavcodec.Packet, name string) string {
	value, ok := packetField[T](packet, name)
	if !ok {
		return "unknown"
	}
	return fmt.Sprint(value)
}

func buildGCSHeartbeatFrame(cfg config) ([]byte, error) {
	builder, err := newCommandFrameBuilder(config{
		Action:            firstNonEmpty(cfg.Action, actionRequestAutopilotVersion),
		SourceSystem:      cfg.SourceSystem,
		SourceComponent:   cfg.SourceComponent,
		TargetSystem:      firstNonZero(cfg.TargetSystem, defaultTargetSystem),
		TargetComponent:   cfg.TargetComponent,
		SimulatorOnly:     true,
		CommandAttempts:   firstNonZero(cfg.CommandAttempts, defaultCommandAttempts),
		RetryInterval:     firstNonZeroDuration(cfg.RetryInterval, defaultCommandRetryInterval),
		LearnRouteSubject: cfg.LearnRouteSubject,
	})
	if err != nil {
		return nil, err
	}
	return builder.GCSHeartbeatFrame()
}

func buildCommandFrame(cfg config) ([]byte, string, error) {
	builder, err := newCommandFrameBuilder(config{
		Route:           cfg.Route,
		Action:          firstNonEmpty(cfg.Action, actionRequestAutopilotVersion),
		SourceSystem:    cfg.SourceSystem,
		SourceComponent: cfg.SourceComponent,
		TargetSystem:    cfg.TargetSystem,
		TargetComponent: cfg.TargetComponent,
		SimulatorOnly:   cfg.SimulatorOnly,
		CommandAttempts: firstNonZero(cfg.CommandAttempts, defaultCommandAttempts),
		RetryInterval:   firstNonZeroDuration(cfg.RetryInterval, defaultCommandRetryInterval),
	})
	if err != nil {
		return nil, "", err
	}
	return builder.CommandFrame(0)
}

func commandForAction(action string) (uint16, [7]float32, error) {
	switch strings.TrimSpace(action) {
	case actionRequestAutopilotVersion:
		return mavcodec.CommandRequestMessage, [7]float32{float32(mavcodec.MessageIDAutopilotVersion)}, nil
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

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func firstNonZeroDuration(values ...time.Duration) time.Duration {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
