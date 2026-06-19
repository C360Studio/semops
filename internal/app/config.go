package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

const (
	EnvNATSURL                    = "SEMOPS_NATS_URL"
	EnvNATSName                   = "SEMOPS_NATS_NAME"
	EnvNATSConnectTimeout         = "SEMOPS_NATS_CONNECT_TIMEOUT"
	EnvAPIAddr                    = "SEMOPS_API_ADDR"
	EnvOwnershipHeartbeatInterval = "SEMOPS_OWNERSHIP_HEARTBEAT_INTERVAL"
	EnvMAVLinkEnabled             = "SEMOPS_MAVLINK_ENABLED"
	EnvMAVLinkSource              = "SEMOPS_MAVLINK_SOURCE"
	EnvOrg                        = "SEMOPS_ORG"
	EnvPlatform                   = "SEMOPS_PLATFORM"
	EnvTraceID                    = "SEMOPS_TRACE_ID"
	EnvMAVLinkWriteTimeout        = "SEMOPS_MAVLINK_WRITE_TIMEOUT"
	EnvMAVLinkUDPListenAddr       = "SEMOPS_MAVLINK_UDP_LISTEN_ADDR"
	EnvMAVLinkUDPMaxDatagramBytes = "SEMOPS_MAVLINK_UDP_MAX_DATAGRAM_BYTES"
	EnvCoTEnabled                 = "SEMOPS_COT_ENABLED"
	EnvCoTSource                  = "SEMOPS_COT_SOURCE"
	EnvCoTWriteTimeout            = "SEMOPS_COT_WRITE_TIMEOUT"
	EnvCoTUDPListenAddr           = "SEMOPS_COT_UDP_LISTEN_ADDR"
	EnvCoTUDPMaxDatagramBytes     = "SEMOPS_COT_UDP_MAX_DATAGRAM_BYTES"
	EnvCoTTCPListenAddr           = "SEMOPS_COT_TCP_LISTEN_ADDR"
	EnvCoTTCPMaxEventBytes        = "SEMOPS_COT_TCP_MAX_EVENT_BYTES"
	EnvCOPGraphQueryTimeout       = "SEMOPS_COP_GRAPH_QUERY_TIMEOUT"
	EnvCOPGraphDiscoveryEnabled   = "SEMOPS_COP_GRAPH_DISCOVERY_ENABLED"
	EnvCOPGraphDiscoveryLimit     = "SEMOPS_COP_GRAPH_DISCOVERY_LIMIT"
	EnvCOPMAVLinkSystemIDs        = "SEMOPS_COP_MAVLINK_SYSTEM_IDS"
	EnvCOPCoTUIDs                 = "SEMOPS_COP_COT_UIDS"
	EnvCOPCAPAlertIDs             = "SEMOPS_COP_CAP_ALERT_IDS"
)

type Config struct {
	NATSURL                    string
	NATSName                   string
	NATSConnectTimeout         time.Duration
	ShutdownTimeout            time.Duration
	APIAddr                    string
	OwnershipHeartbeatInterval time.Duration
	MAVLink                    MAVLinkConfig
	CoT                        CoTConfig
	COP                        COPConfig
}

type MAVLinkConfig struct {
	Enabled       bool
	Source        string
	Org           string
	Platform      string
	TraceID       string
	RawMaxRecords int
	RawMaxBytes   int
	WriteTimeout  time.Duration
	Retry         natsclient.RetryConfig
	UDP           MAVLinkUDPConfig
}

type MAVLinkUDPConfig struct {
	ListenAddr       string
	MaxDatagramBytes int
}

type CoTConfig struct {
	Enabled       bool
	Source        string
	Org           string
	Platform      string
	TraceID       string
	RawMaxRecords int
	RawMaxBytes   int
	WriteTimeout  time.Duration
	Retry         natsclient.RetryConfig
	UDP           CoTUDPConfig
	TCP           CoTTCPConfig
}

type CoTUDPConfig struct {
	ListenAddr       string
	MaxDatagramBytes int
}

type CoTTCPConfig struct {
	ListenAddr    string
	MaxEventBytes int
}

type COPConfig struct {
	GraphQueryTimeout     time.Duration
	GraphDiscoveryEnabled bool
	GraphDiscoveryLimit   int
	MAVLinkSystemIDs      []int
	CoTUIDs               []string
	CAPAlertIDs           []string
}

func DefaultConfig() Config {
	return Config{
		NATSURL:                    "nats://127.0.0.1:4222",
		NATSName:                   "semops",
		NATSConnectTimeout:         5 * time.Second,
		ShutdownTimeout:            2 * time.Second,
		APIAddr:                    ":8088",
		OwnershipHeartbeatInterval: ownership.HeartbeatInterval,
		MAVLink: MAVLinkConfig{
			Enabled:      true,
			Source:       "mavlink:inprocess",
			Org:          "c360",
			Platform:     "edge",
			TraceID:      "semops-mavlink-hosted",
			WriteTimeout: 2 * time.Second,
			UDP: MAVLinkUDPConfig{
				MaxDatagramBytes: 4096,
			},
			Retry: natsclient.RetryConfig{
				MaxRetries:        5,
				InitialBackoff:    50 * time.Millisecond,
				MaxBackoff:        500 * time.Millisecond,
				BackoffMultiplier: 2,
			},
		},
		CoT: CoTConfig{
			Enabled:      false,
			Source:       "tak-cot:inprocess",
			Org:          "c360",
			Platform:     "edge",
			TraceID:      "semops-cot-hosted",
			WriteTimeout: 2 * time.Second,
			UDP: CoTUDPConfig{
				MaxDatagramBytes: 64 * 1024,
			},
			TCP: CoTTCPConfig{
				MaxEventBytes: 1024 * 1024,
			},
			Retry: natsclient.RetryConfig{
				MaxRetries:        5,
				InitialBackoff:    50 * time.Millisecond,
				MaxBackoff:        500 * time.Millisecond,
				BackoffMultiplier: 2,
			},
		},
		COP: COPConfig{
			GraphQueryTimeout:     2 * time.Second,
			GraphDiscoveryEnabled: true,
			GraphDiscoveryLimit:   500,
			MAVLinkSystemIDs:      []int{42},
			CoTUIDs: []string{
				"ANDROID-ALPHA",
				"ANDROID-BRAVO",
				"MARKER-NORTH-GATE",
				"CHAT-ALPHA-1",
			},
			CAPAlertIDs: []string{
				"nws-demo-flood-warning",
			},
		},
	}
}

func ConfigFromEnv(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	cfg := DefaultConfig()

	setString(getenv, EnvNATSURL, &cfg.NATSURL)
	setString(getenv, EnvNATSName, &cfg.NATSName)
	setString(getenv, EnvAPIAddr, &cfg.APIAddr)
	setString(getenv, EnvMAVLinkSource, &cfg.MAVLink.Source)
	setString(getenv, EnvCoTSource, &cfg.CoT.Source)
	setString(getenv, EnvOrg, &cfg.MAVLink.Org)
	setString(getenv, EnvOrg, &cfg.CoT.Org)
	setString(getenv, EnvPlatform, &cfg.MAVLink.Platform)
	setString(getenv, EnvPlatform, &cfg.CoT.Platform)
	setString(getenv, EnvTraceID, &cfg.MAVLink.TraceID)
	setString(getenv, EnvTraceID, &cfg.CoT.TraceID)
	setString(getenv, EnvMAVLinkUDPListenAddr, &cfg.MAVLink.UDP.ListenAddr)
	setString(getenv, EnvCoTUDPListenAddr, &cfg.CoT.UDP.ListenAddr)
	setString(getenv, EnvCoTTCPListenAddr, &cfg.CoT.TCP.ListenAddr)

	var err error
	if cfg.NATSConnectTimeout, err = durationFromEnv(getenv, EnvNATSConnectTimeout, cfg.NATSConnectTimeout); err != nil {
		return Config{}, err
	}
	if cfg.OwnershipHeartbeatInterval, err = durationFromEnv(
		getenv,
		EnvOwnershipHeartbeatInterval,
		cfg.OwnershipHeartbeatInterval,
	); err != nil {
		return Config{}, err
	}
	if cfg.MAVLink.WriteTimeout, err = durationFromEnv(
		getenv,
		EnvMAVLinkWriteTimeout,
		cfg.MAVLink.WriteTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.CoT.WriteTimeout, err = durationFromEnv(
		getenv,
		EnvCoTWriteTimeout,
		cfg.CoT.WriteTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.COP.GraphQueryTimeout, err = durationFromEnv(
		getenv,
		EnvCOPGraphQueryTimeout,
		cfg.COP.GraphQueryTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.COP.GraphDiscoveryEnabled, err = boolFromEnv(
		getenv,
		EnvCOPGraphDiscoveryEnabled,
		cfg.COP.GraphDiscoveryEnabled,
	); err != nil {
		return Config{}, err
	}
	if cfg.MAVLink.Enabled, err = boolFromEnv(getenv, EnvMAVLinkEnabled, cfg.MAVLink.Enabled); err != nil {
		return Config{}, err
	}
	if cfg.CoT.Enabled, err = boolFromEnv(getenv, EnvCoTEnabled, cfg.CoT.Enabled); err != nil {
		return Config{}, err
	}
	if cfg.MAVLink.UDP.MaxDatagramBytes, err = intFromEnv(
		getenv,
		EnvMAVLinkUDPMaxDatagramBytes,
		cfg.MAVLink.UDP.MaxDatagramBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.CoT.UDP.MaxDatagramBytes, err = intFromEnv(
		getenv,
		EnvCoTUDPMaxDatagramBytes,
		cfg.CoT.UDP.MaxDatagramBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.CoT.TCP.MaxEventBytes, err = intFromEnv(
		getenv,
		EnvCoTTCPMaxEventBytes,
		cfg.CoT.TCP.MaxEventBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.COP.GraphDiscoveryLimit, err = intFromEnv(
		getenv,
		EnvCOPGraphDiscoveryLimit,
		cfg.COP.GraphDiscoveryLimit,
	); err != nil {
		return Config{}, err
	}
	if cfg.COP.MAVLinkSystemIDs, err = intListFromEnv(getenv, EnvCOPMAVLinkSystemIDs, cfg.COP.MAVLinkSystemIDs); err != nil {
		return Config{}, err
	}
	if cfg.COP.CoTUIDs, err = stringListFromEnv(getenv, EnvCOPCoTUIDs, cfg.COP.CoTUIDs); err != nil {
		return Config{}, err
	}
	if cfg.COP.CAPAlertIDs, err = stringListFromEnv(getenv, EnvCOPCAPAlertIDs, cfg.COP.CAPAlertIDs); err != nil {
		return Config{}, err
	}

	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.NATSURL) == "" {
		return fmt.Errorf("%s is required", EnvNATSURL)
	}
	if strings.TrimSpace(c.NATSName) == "" {
		return fmt.Errorf("%s is required", EnvNATSName)
	}
	if c.NATSConnectTimeout <= 0 {
		return fmt.Errorf("%s must be greater than zero", EnvNATSConnectTimeout)
	}
	if strings.TrimSpace(c.APIAddr) == "" {
		return fmt.Errorf("%s is required", EnvAPIAddr)
	}
	if c.COP.GraphQueryTimeout <= 0 {
		return fmt.Errorf("%s must be greater than zero", EnvCOPGraphQueryTimeout)
	}
	if c.COP.GraphDiscoveryLimit <= 0 {
		return fmt.Errorf("%s must be greater than zero", EnvCOPGraphDiscoveryLimit)
	}
	if len(c.COP.MAVLinkSystemIDs) == 0 {
		return fmt.Errorf("%s must include at least one system id", EnvCOPMAVLinkSystemIDs)
	}
	for _, systemID := range c.COP.MAVLinkSystemIDs {
		if systemID < 0 || systemID > 255 {
			return fmt.Errorf("%s contains invalid MAVLink system id %d", EnvCOPMAVLinkSystemIDs, systemID)
		}
	}
	if len(c.COP.CoTUIDs) == 0 {
		return fmt.Errorf("%s must include at least one UID", EnvCOPCoTUIDs)
	}
	if len(c.COP.CAPAlertIDs) == 0 {
		return fmt.Errorf("%s must include at least one alert identifier", EnvCOPCAPAlertIDs)
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be greater than zero")
	}
	if c.OwnershipHeartbeatInterval <= 0 {
		return fmt.Errorf("%s must be greater than zero", EnvOwnershipHeartbeatInterval)
	}
	if err := c.CoT.Validate(); err != nil {
		return err
	}
	if !c.MAVLink.Enabled {
		return nil
	}
	if strings.TrimSpace(c.MAVLink.Source) == "" {
		return fmt.Errorf("%s is required when MAVLink is enabled", EnvMAVLinkSource)
	}
	if strings.TrimSpace(c.MAVLink.Org) == "" {
		return fmt.Errorf("%s is required when MAVLink is enabled", EnvOrg)
	}
	if strings.TrimSpace(c.MAVLink.Platform) == "" {
		return fmt.Errorf("%s is required when MAVLink is enabled", EnvPlatform)
	}
	if c.MAVLink.WriteTimeout <= 0 {
		return fmt.Errorf("%s must be greater than zero when MAVLink is enabled", EnvMAVLinkWriteTimeout)
	}
	if strings.TrimSpace(c.MAVLink.UDP.ListenAddr) != "" && c.MAVLink.UDP.MaxDatagramBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when MAVLink UDP is enabled", EnvMAVLinkUDPMaxDatagramBytes)
	}
	return nil
}

func (c CoTConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Source) == "" {
		return fmt.Errorf("%s is required when CoT is enabled", EnvCoTSource)
	}
	if strings.TrimSpace(c.Org) == "" {
		return fmt.Errorf("%s is required when CoT is enabled", EnvOrg)
	}
	if strings.TrimSpace(c.Platform) == "" {
		return fmt.Errorf("%s is required when CoT is enabled", EnvPlatform)
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("%s must be greater than zero when CoT is enabled", EnvCoTWriteTimeout)
	}
	if strings.TrimSpace(c.UDP.ListenAddr) != "" && c.UDP.MaxDatagramBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when CoT UDP is enabled", EnvCoTUDPMaxDatagramBytes)
	}
	if strings.TrimSpace(c.TCP.ListenAddr) != "" && c.TCP.MaxEventBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when CoT TCP is enabled", EnvCoTTCPMaxEventBytes)
	}
	return nil
}

func setString(getenv func(string) string, name string, target *string) {
	if value := strings.TrimSpace(getenv(name)); value != "" {
		*target = value
	}
}

func durationFromEnv(
	getenv func(string) string,
	name string,
	fallback time.Duration,
) (time.Duration, error) {
	value := strings.TrimSpace(getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func boolFromEnv(getenv func(string) string, name string, fallback bool) (bool, error) {
	value := strings.TrimSpace(getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func intFromEnv(getenv func(string) string, name string, fallback int) (int, error) {
	value := strings.TrimSpace(getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func intListFromEnv(getenv func(string) string, name string, fallback []int) ([]int, error) {
	value := strings.TrimSpace(getenv(name))
	if value == "" {
		return append([]int(nil), fallback...), nil
	}
	parts := strings.Split(value, ",")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parsed, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		values = append(values, parsed)
	}
	return values, nil
}

func stringListFromEnv(getenv func(string) string, name string, fallback []string) ([]string, error) {
	value := strings.TrimSpace(getenv(name))
	if value == "" {
		return append([]string(nil), fallback...), nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		values = append(values, part)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("parse %s: empty list", name)
	}
	return values, nil
}
