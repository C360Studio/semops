package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	adsbcomponent "github.com/c360studio/semops/internal/components/adsb"
	capcomponent "github.com/c360studio/semops/internal/components/cap"
	fusioncomponent "github.com/c360studio/semops/internal/components/fusion"
	klvcomponent "github.com/c360studio/semops/internal/components/klv"
	sapientcomponent "github.com/c360studio/semops/internal/components/sapient"
	weathercomponent "github.com/c360studio/semops/internal/components/weather"
	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	sapientcodec "github.com/c360studio/semops/pkg/adapters/sapient"
	weathercodec "github.com/c360studio/semops/pkg/adapters/weather"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

const (
	EnvNATSURL                      = "SEMOPS_NATS_URL"
	EnvNATSName                     = "SEMOPS_NATS_NAME"
	EnvNATSConnectTimeout           = "SEMOPS_NATS_CONNECT_TIMEOUT"
	EnvAPIAddr                      = "SEMOPS_API_ADDR"
	EnvOwnershipHeartbeatInterval   = "SEMOPS_OWNERSHIP_HEARTBEAT_INTERVAL"
	EnvMAVLinkEnabled               = "SEMOPS_MAVLINK_ENABLED"
	EnvMAVLinkSource                = "SEMOPS_MAVLINK_SOURCE"
	EnvOrg                          = "SEMOPS_ORG"
	EnvPlatform                     = "SEMOPS_PLATFORM"
	EnvTraceID                      = "SEMOPS_TRACE_ID"
	EnvMAVLinkWriteTimeout          = "SEMOPS_MAVLINK_WRITE_TIMEOUT"
	EnvMAVLinkUDPListenAddr         = "SEMOPS_MAVLINK_UDP_LISTEN_ADDR"
	EnvMAVLinkUDPMaxDatagramBytes   = "SEMOPS_MAVLINK_UDP_MAX_DATAGRAM_BYTES"
	EnvCoTEnabled                   = "SEMOPS_COT_ENABLED"
	EnvCoTSource                    = "SEMOPS_COT_SOURCE"
	EnvCoTWriteTimeout              = "SEMOPS_COT_WRITE_TIMEOUT"
	EnvCoTUDPListenAddr             = "SEMOPS_COT_UDP_LISTEN_ADDR"
	EnvCoTUDPMaxDatagramBytes       = "SEMOPS_COT_UDP_MAX_DATAGRAM_BYTES"
	EnvCoTTCPListenAddr             = "SEMOPS_COT_TCP_LISTEN_ADDR"
	EnvCoTTCPMaxEventBytes          = "SEMOPS_COT_TCP_MAX_EVENT_BYTES"
	EnvCAPEnabled                   = "SEMOPS_CAP_ENABLED"
	EnvCAPSource                    = "SEMOPS_CAP_SOURCE"
	EnvCAPReplayPath                = "SEMOPS_CAP_REPLAY_PATH"
	EnvCAPWriteTimeout              = "SEMOPS_CAP_WRITE_TIMEOUT"
	EnvCAPHTTPURL                   = "SEMOPS_CAP_HTTP_URL"
	EnvCAPHTTPMethod                = "SEMOPS_CAP_HTTP_METHOD"
	EnvCAPHTTPPollInterval          = "SEMOPS_CAP_HTTP_POLL_INTERVAL"
	EnvCAPHTTPStaleAfter            = "SEMOPS_CAP_HTTP_STALE_AFTER"
	EnvCAPHTTPContactPolicy         = "SEMOPS_CAP_HTTP_CONTACT_POLICY"
	EnvCAPHTTPAuthRef               = "SEMOPS_CAP_HTTP_AUTH_REF"
	EnvCAPHTTPMaxResponseBytes      = "SEMOPS_CAP_HTTP_MAX_RESPONSE_BYTES"
	EnvADSBEnabled                  = "SEMOPS_ADSB_ENABLED"
	EnvADSBSource                   = "SEMOPS_ADSB_SOURCE"
	EnvADSBReplayPath               = "SEMOPS_ADSB_REPLAY_PATH"
	EnvADSBRawMaxRecords            = "SEMOPS_ADSB_RAW_MAX_RECORDS"
	EnvADSBRawMaxBytes              = "SEMOPS_ADSB_RAW_MAX_BYTES"
	EnvADSBWriteTimeout             = "SEMOPS_ADSB_WRITE_TIMEOUT"
	EnvADSBHTTPURL                  = "SEMOPS_ADSB_HTTP_URL"
	EnvADSBHTTPMethod               = "SEMOPS_ADSB_HTTP_METHOD"
	EnvADSBHTTPPollInterval         = "SEMOPS_ADSB_HTTP_POLL_INTERVAL"
	EnvADSBHTTPStaleAfter           = "SEMOPS_ADSB_HTTP_STALE_AFTER"
	EnvADSBHTTPContactPolicy        = "SEMOPS_ADSB_HTTP_CONTACT_POLICY"
	EnvADSBHTTPAuthRef              = "SEMOPS_ADSB_HTTP_AUTH_REF"
	EnvADSBHTTPMaxResponseBytes     = "SEMOPS_ADSB_HTTP_MAX_RESPONSE_BYTES"
	EnvSAPIENTEnabled               = "SEMOPS_SAPIENT_ENABLED"
	EnvSAPIENTGraphEnabled          = "SEMOPS_SAPIENT_GRAPH_ENABLED"
	EnvSAPIENTSource                = "SEMOPS_SAPIENT_SOURCE"
	EnvSAPIENTReplayPath            = "SEMOPS_SAPIENT_REPLAY_PATH"
	EnvSAPIENTRawMaxRecords         = "SEMOPS_SAPIENT_RAW_MAX_RECORDS"
	EnvSAPIENTRawMaxBytes           = "SEMOPS_SAPIENT_RAW_MAX_BYTES"
	EnvSAPIENTWriteTimeout          = "SEMOPS_SAPIENT_WRITE_TIMEOUT"
	EnvSAPIENTHTTPURL               = "SEMOPS_SAPIENT_HTTP_URL"
	EnvSAPIENTHTTPMethod            = "SEMOPS_SAPIENT_HTTP_METHOD"
	EnvSAPIENTHTTPPollInterval      = "SEMOPS_SAPIENT_HTTP_POLL_INTERVAL"
	EnvSAPIENTHTTPStaleAfter        = "SEMOPS_SAPIENT_HTTP_STALE_AFTER"
	EnvSAPIENTHTTPContactPolicy     = "SEMOPS_SAPIENT_HTTP_CONTACT_POLICY"
	EnvSAPIENTHTTPAuthRef           = "SEMOPS_SAPIENT_HTTP_AUTH_REF"
	EnvSAPIENTHTTPMaxResponseBytes  = "SEMOPS_SAPIENT_HTTP_MAX_RESPONSE_BYTES"
	EnvSAPIENTHTTPEncoding          = "SEMOPS_SAPIENT_HTTP_ENCODING"
	EnvKLVEnabled                   = "SEMOPS_KLV_ENABLED"
	EnvKLVSource                    = "SEMOPS_KLV_SOURCE"
	EnvKLVMediaPath                 = "SEMOPS_KLV_MEDIA_PATH"
	EnvKLVMediaPattern              = "SEMOPS_KLV_MEDIA_PATTERN"
	EnvKLVWriteTimeout              = "SEMOPS_KLV_WRITE_TIMEOUT"
	EnvKLVDemuxMaxPacketBytes       = "SEMOPS_KLV_DEMUX_MAX_PACKET_BYTES"
	EnvKLVDemuxMaxExtractBytes      = "SEMOPS_KLV_DEMUX_MAX_EXTRACT_BYTES"
	EnvKLVDemuxMaxPackets           = "SEMOPS_KLV_DEMUX_MAX_PACKETS"
	EnvKLVDemuxMaxMaterializedBytes = "SEMOPS_KLV_DEMUX_MAX_MATERIALIZED_BYTES"
	EnvKLVDemuxProbeOutputMaxBytes  = "SEMOPS_KLV_DEMUX_PROBE_OUTPUT_MAX_BYTES"
	EnvKLVDecodeMaxPacketBytes      = "SEMOPS_KLV_DECODE_MAX_PACKET_BYTES"
	EnvWeatherEnabled               = "SEMOPS_WEATHER_ENABLED"
	EnvWeatherSource                = "SEMOPS_WEATHER_SOURCE"
	EnvWeatherProvider              = "SEMOPS_WEATHER_PROVIDER"
	EnvWeatherQueryShape            = "SEMOPS_WEATHER_QUERY_SHAPE"
	EnvWeatherFixturePath           = "SEMOPS_WEATHER_FIXTURE_PATH"
	EnvWeatherWriteTimeout          = "SEMOPS_WEATHER_WRITE_TIMEOUT"
	EnvWeatherFreshness             = "SEMOPS_WEATHER_FRESHNESS"
	EnvWeatherMaxObservations       = "SEMOPS_WEATHER_MAX_OBSERVATIONS"
	EnvFusionEnabled                = "SEMOPS_FUSION_ENABLED"
	EnvFusionCandidateSubject       = "SEMOPS_FUSION_CANDIDATE_SUBJECT"
	EnvFusionWriteTimeout           = "SEMOPS_FUSION_WRITE_TIMEOUT"
	EnvCOPGraphQueryTimeout         = "SEMOPS_COP_GRAPH_QUERY_TIMEOUT"
	EnvCOPGraphDiscoveryEnabled     = "SEMOPS_COP_GRAPH_DISCOVERY_ENABLED"
	EnvCOPGraphDiscoveryLimit       = "SEMOPS_COP_GRAPH_DISCOVERY_LIMIT"
	EnvCOPMAVLinkSystemIDs          = "SEMOPS_COP_MAVLINK_SYSTEM_IDS"
	EnvCOPCoTUIDs                   = "SEMOPS_COP_COT_UIDS"
	EnvCOPCAPAlertIDs               = "SEMOPS_COP_CAP_ALERT_IDS"
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
	CAP                        CAPConfig
	ADSB                       ADSBConfig
	SAPIENT                    SAPIENTConfig
	KLV                        KLVConfig
	Weather                    WeatherConfig
	Fusion                     FusionConfig
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

type CAPConfig struct {
	Enabled      bool
	Source       string
	Org          string
	Platform     string
	TraceID      string
	ReplayPath   string
	WriteTimeout time.Duration
	Retry        natsclient.RetryConfig
	HTTP         CAPHTTPConfig
}

type CAPHTTPConfig struct {
	URL              string
	Method           string
	PollInterval     time.Duration
	StaleAfter       time.Duration
	ContactPolicy    string
	AuthRef          string
	MaxResponseBytes int
}

type ADSBConfig struct {
	Enabled       bool
	Source        string
	Org           string
	Platform      string
	TraceID       string
	ReplayPath    string
	RawMaxRecords int
	RawMaxBytes   int
	WriteTimeout  time.Duration
	Retry         natsclient.RetryConfig
	HTTP          ADSBHTTPConfig
}

type ADSBHTTPConfig struct {
	URL              string
	Method           string
	PollInterval     time.Duration
	StaleAfter       time.Duration
	ContactPolicy    string
	AuthRef          string
	MaxResponseBytes int
}

type SAPIENTConfig struct {
	Enabled       bool
	GraphEnabled  bool
	Source        string
	Org           string
	Platform      string
	TraceID       string
	ReplayPath    string
	RawMaxRecords int
	RawMaxBytes   int
	WriteTimeout  time.Duration
	Retry         natsclient.RetryConfig
	HTTP          SAPIENTHTTPConfig
}

type SAPIENTHTTPConfig struct {
	URL              string
	Method           string
	PollInterval     time.Duration
	StaleAfter       time.Duration
	ContactPolicy    string
	AuthRef          string
	MaxResponseBytes int
	Encoding         sapientcodec.Encoding
}

type KLVConfig struct {
	Enabled      bool
	Source       string
	Org          string
	Platform     string
	TraceID      string
	MediaPath    string
	MediaPattern string
	WriteTimeout time.Duration
	Retry        natsclient.RetryConfig
	Demux        KLVDemuxConfig
	Decode       KLVDecodeConfig
}

type KLVDemuxConfig struct {
	MaxPacketBytes       int
	MaxExtractBytes      int
	MaxPackets           int
	MaxMaterializedBytes int
	ProbeOutputMaxBytes  int
}

type KLVDecodeConfig struct {
	MaxPacketBytes int
}

type WeatherConfig struct {
	Enabled         bool
	Source          string
	Org             string
	Platform        string
	TraceID         string
	Provider        string
	QueryShape      string
	FixturePath     string
	WriteTimeout    time.Duration
	Freshness       time.Duration
	MaxObservations int
	Retry           natsclient.RetryConfig
}

type FusionConfig struct {
	Enabled          bool
	Org              string
	Platform         string
	TraceID          string
	CandidateSubject string
	WriteTimeout     time.Duration
	Retry            natsclient.RetryConfig
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
		CAP: CAPConfig{
			Enabled:      false,
			Source:       "cap:http:inprocess",
			Org:          "c360",
			Platform:     "edge",
			TraceID:      "semops-cap-hosted",
			WriteTimeout: 2 * time.Second,
			HTTP: CAPHTTPConfig{
				URL:              capcomponent.DefaultHTTPPollURL,
				Method:           "GET",
				PollInterval:     capcomponent.DefaultHTTPPollInterval,
				StaleAfter:       time.Duration(capcomponent.DefaultHTTPStaleMultiplier) * capcomponent.DefaultHTTPPollInterval,
				MaxResponseBytes: capcomponent.DefaultHTTPMaxResponseBytes,
			},
			Retry: natsclient.RetryConfig{
				MaxRetries:        5,
				InitialBackoff:    50 * time.Millisecond,
				MaxBackoff:        500 * time.Millisecond,
				BackoffMultiplier: 2,
			},
		},
		ADSB: ADSBConfig{
			Enabled:       false,
			Source:        "adsb:opensky:inprocess",
			Org:           "c360",
			Platform:      "edge",
			TraceID:       "semops-adsb-hosted",
			RawMaxRecords: adsbcodec.DefaultRawLaneMaxRecords,
			RawMaxBytes:   adsbcodec.DefaultRawLaneMaxBytes,
			WriteTimeout:  2 * time.Second,
			HTTP: ADSBHTTPConfig{
				URL:              adsbcomponent.DefaultOpenSkyPollURL,
				Method:           "GET",
				PollInterval:     adsbcomponent.DefaultHTTPPollInterval,
				StaleAfter:       time.Duration(adsbcomponent.DefaultHTTPStaleMultiplier) * adsbcomponent.DefaultHTTPPollInterval,
				MaxResponseBytes: adsbcomponent.DefaultHTTPMaxResponseBytes,
			},
			Retry: natsclient.RetryConfig{
				MaxRetries:        5,
				InitialBackoff:    50 * time.Millisecond,
				MaxBackoff:        500 * time.Millisecond,
				BackoffMultiplier: 2,
			},
		},
		SAPIENT: SAPIENTConfig{
			Enabled:       false,
			GraphEnabled:  false,
			Source:        "sapient:http:inprocess",
			Org:           "c360",
			Platform:      "edge",
			TraceID:       "semops-sapient-hosted",
			RawMaxRecords: sapientcodec.DefaultRawLaneMaxRecords,
			RawMaxBytes:   sapientcodec.DefaultRawLaneMaxBytes,
			WriteTimeout:  2 * time.Second,
			HTTP: SAPIENTHTTPConfig{
				Method:           "GET",
				PollInterval:     sapientcomponent.DefaultHTTPPollInterval,
				StaleAfter:       time.Duration(sapientcomponent.DefaultHTTPStaleMultiplier) * sapientcomponent.DefaultHTTPPollInterval,
				MaxResponseBytes: sapientcomponent.DefaultHTTPMaxResponseBytes,
			},
			Retry: natsclient.RetryConfig{
				MaxRetries:        5,
				InitialBackoff:    50 * time.Millisecond,
				MaxBackoff:        500 * time.Millisecond,
				BackoffMultiplier: 2,
			},
		},
		KLV: KLVConfig{
			Enabled:      false,
			Source:       "klv:media-ref:inprocess",
			Org:          "c360",
			Platform:     "edge",
			TraceID:      "semops-klv-hosted",
			MediaPath:    klvcomponent.DefaultMediaPath,
			MediaPattern: klvcomponent.DefaultMediaPattern,
			WriteTimeout: 2 * time.Second,
			Demux: KLVDemuxConfig{
				MaxPacketBytes:       klvcomponent.DefaultMaxPacketBytes,
				MaxExtractBytes:      klvcomponent.DefaultMaxPacketBytes * 4,
				MaxPackets:           klvcomponent.DefaultMaxPackets,
				MaxMaterializedBytes: klvcomponent.DefaultMaxMaterializedBytes,
				ProbeOutputMaxBytes:  klvcomponent.DefaultProbeOutputMaxBytes,
			},
			Decode: KLVDecodeConfig{
				MaxPacketBytes: klvcomponent.DefaultMaxMaterializedPacketBytes,
			},
			Retry: natsclient.RetryConfig{
				MaxRetries:        5,
				InitialBackoff:    50 * time.Millisecond,
				MaxBackoff:        500 * time.Millisecond,
				BackoffMultiplier: 2,
			},
		},
		Weather: WeatherConfig{
			Enabled:         false,
			Source:          "weather:fixture:inprocess",
			Org:             "c360",
			Platform:        "edge",
			TraceID:         "semops-weather-hosted",
			Provider:        weathercodec.ProviderOpenMeteo,
			QueryShape:      weathercodec.QueryShapePosition,
			FixturePath:     weathercomponent.DefaultFixturePath,
			WriteTimeout:    2 * time.Second,
			Freshness:       weathercomponent.DefaultFreshness,
			MaxObservations: weathercomponent.DefaultMaxObservations,
			Retry: natsclient.RetryConfig{
				MaxRetries:        5,
				InitialBackoff:    50 * time.Millisecond,
				MaxBackoff:        500 * time.Millisecond,
				BackoffMultiplier: 2,
			},
		},
		Fusion: FusionConfig{
			Enabled:          false,
			Org:              "c360",
			Platform:         "edge",
			TraceID:          "semops-fusion-hosted",
			CandidateSubject: fusioncomponent.DefaultCandidateSubject,
			WriteTimeout:     2 * time.Second,
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
	setString(getenv, EnvCAPSource, &cfg.CAP.Source)
	setString(getenv, EnvCAPReplayPath, &cfg.CAP.ReplayPath)
	setString(getenv, EnvADSBSource, &cfg.ADSB.Source)
	setString(getenv, EnvADSBReplayPath, &cfg.ADSB.ReplayPath)
	setString(getenv, EnvSAPIENTSource, &cfg.SAPIENT.Source)
	setString(getenv, EnvSAPIENTReplayPath, &cfg.SAPIENT.ReplayPath)
	setString(getenv, EnvKLVSource, &cfg.KLV.Source)
	setString(getenv, EnvKLVMediaPath, &cfg.KLV.MediaPath)
	setString(getenv, EnvKLVMediaPattern, &cfg.KLV.MediaPattern)
	setString(getenv, EnvWeatherSource, &cfg.Weather.Source)
	setString(getenv, EnvWeatherProvider, &cfg.Weather.Provider)
	setString(getenv, EnvWeatherQueryShape, &cfg.Weather.QueryShape)
	setString(getenv, EnvWeatherFixturePath, &cfg.Weather.FixturePath)
	setString(getenv, EnvFusionCandidateSubject, &cfg.Fusion.CandidateSubject)
	setString(getenv, EnvOrg, &cfg.MAVLink.Org)
	setString(getenv, EnvOrg, &cfg.CoT.Org)
	setString(getenv, EnvOrg, &cfg.CAP.Org)
	setString(getenv, EnvOrg, &cfg.ADSB.Org)
	setString(getenv, EnvOrg, &cfg.SAPIENT.Org)
	setString(getenv, EnvOrg, &cfg.KLV.Org)
	setString(getenv, EnvOrg, &cfg.Weather.Org)
	setString(getenv, EnvOrg, &cfg.Fusion.Org)
	setString(getenv, EnvPlatform, &cfg.MAVLink.Platform)
	setString(getenv, EnvPlatform, &cfg.CoT.Platform)
	setString(getenv, EnvPlatform, &cfg.CAP.Platform)
	setString(getenv, EnvPlatform, &cfg.ADSB.Platform)
	setString(getenv, EnvPlatform, &cfg.SAPIENT.Platform)
	setString(getenv, EnvPlatform, &cfg.KLV.Platform)
	setString(getenv, EnvPlatform, &cfg.Weather.Platform)
	setString(getenv, EnvPlatform, &cfg.Fusion.Platform)
	setString(getenv, EnvTraceID, &cfg.MAVLink.TraceID)
	setString(getenv, EnvTraceID, &cfg.CoT.TraceID)
	setString(getenv, EnvTraceID, &cfg.CAP.TraceID)
	setString(getenv, EnvTraceID, &cfg.ADSB.TraceID)
	setString(getenv, EnvTraceID, &cfg.SAPIENT.TraceID)
	setString(getenv, EnvTraceID, &cfg.KLV.TraceID)
	setString(getenv, EnvTraceID, &cfg.Weather.TraceID)
	setString(getenv, EnvTraceID, &cfg.Fusion.TraceID)
	setString(getenv, EnvMAVLinkUDPListenAddr, &cfg.MAVLink.UDP.ListenAddr)
	setString(getenv, EnvCoTUDPListenAddr, &cfg.CoT.UDP.ListenAddr)
	setString(getenv, EnvCoTTCPListenAddr, &cfg.CoT.TCP.ListenAddr)
	setString(getenv, EnvCAPHTTPURL, &cfg.CAP.HTTP.URL)
	setString(getenv, EnvCAPHTTPMethod, &cfg.CAP.HTTP.Method)
	setString(getenv, EnvCAPHTTPContactPolicy, &cfg.CAP.HTTP.ContactPolicy)
	setString(getenv, EnvCAPHTTPAuthRef, &cfg.CAP.HTTP.AuthRef)
	setString(getenv, EnvADSBHTTPURL, &cfg.ADSB.HTTP.URL)
	setString(getenv, EnvADSBHTTPMethod, &cfg.ADSB.HTTP.Method)
	setString(getenv, EnvADSBHTTPContactPolicy, &cfg.ADSB.HTTP.ContactPolicy)
	setString(getenv, EnvADSBHTTPAuthRef, &cfg.ADSB.HTTP.AuthRef)
	setString(getenv, EnvSAPIENTHTTPURL, &cfg.SAPIENT.HTTP.URL)
	setString(getenv, EnvSAPIENTHTTPMethod, &cfg.SAPIENT.HTTP.Method)
	setString(getenv, EnvSAPIENTHTTPContactPolicy, &cfg.SAPIENT.HTTP.ContactPolicy)
	setString(getenv, EnvSAPIENTHTTPAuthRef, &cfg.SAPIENT.HTTP.AuthRef)
	if value := strings.TrimSpace(getenv(EnvSAPIENTHTTPEncoding)); value != "" {
		cfg.SAPIENT.HTTP.Encoding = sapientcodec.Encoding(value)
	}

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
	if cfg.CAP.WriteTimeout, err = durationFromEnv(
		getenv,
		EnvCAPWriteTimeout,
		cfg.CAP.WriteTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.ADSB.WriteTimeout, err = durationFromEnv(
		getenv,
		EnvADSBWriteTimeout,
		cfg.ADSB.WriteTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.SAPIENT.WriteTimeout, err = durationFromEnv(
		getenv,
		EnvSAPIENTWriteTimeout,
		cfg.SAPIENT.WriteTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.KLV.WriteTimeout, err = durationFromEnv(
		getenv,
		EnvKLVWriteTimeout,
		cfg.KLV.WriteTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.Weather.WriteTimeout, err = durationFromEnv(
		getenv,
		EnvWeatherWriteTimeout,
		cfg.Weather.WriteTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.Fusion.WriteTimeout, err = durationFromEnv(
		getenv,
		EnvFusionWriteTimeout,
		cfg.Fusion.WriteTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.Weather.Freshness, err = durationFromEnv(
		getenv,
		EnvWeatherFreshness,
		cfg.Weather.Freshness,
	); err != nil {
		return Config{}, err
	}
	if cfg.CAP.HTTP.PollInterval, err = durationFromEnv(
		getenv,
		EnvCAPHTTPPollInterval,
		cfg.CAP.HTTP.PollInterval,
	); err != nil {
		return Config{}, err
	}
	if cfg.CAP.HTTP.StaleAfter, err = durationFromEnv(
		getenv,
		EnvCAPHTTPStaleAfter,
		cfg.CAP.HTTP.StaleAfter,
	); err != nil {
		return Config{}, err
	}
	if cfg.ADSB.HTTP.PollInterval, err = durationFromEnv(
		getenv,
		EnvADSBHTTPPollInterval,
		cfg.ADSB.HTTP.PollInterval,
	); err != nil {
		return Config{}, err
	}
	if cfg.ADSB.HTTP.StaleAfter, err = durationFromEnv(
		getenv,
		EnvADSBHTTPStaleAfter,
		cfg.ADSB.HTTP.StaleAfter,
	); err != nil {
		return Config{}, err
	}
	if cfg.SAPIENT.HTTP.PollInterval, err = durationFromEnv(
		getenv,
		EnvSAPIENTHTTPPollInterval,
		cfg.SAPIENT.HTTP.PollInterval,
	); err != nil {
		return Config{}, err
	}
	if cfg.SAPIENT.HTTP.StaleAfter, err = durationFromEnv(
		getenv,
		EnvSAPIENTHTTPStaleAfter,
		cfg.SAPIENT.HTTP.StaleAfter,
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
	if cfg.CAP.Enabled, err = boolFromEnv(getenv, EnvCAPEnabled, cfg.CAP.Enabled); err != nil {
		return Config{}, err
	}
	if cfg.ADSB.Enabled, err = boolFromEnv(getenv, EnvADSBEnabled, cfg.ADSB.Enabled); err != nil {
		return Config{}, err
	}
	if cfg.SAPIENT.Enabled, err = boolFromEnv(getenv, EnvSAPIENTEnabled, cfg.SAPIENT.Enabled); err != nil {
		return Config{}, err
	}
	if cfg.SAPIENT.GraphEnabled, err = boolFromEnv(
		getenv,
		EnvSAPIENTGraphEnabled,
		cfg.SAPIENT.GraphEnabled,
	); err != nil {
		return Config{}, err
	}
	if cfg.KLV.Enabled, err = boolFromEnv(getenv, EnvKLVEnabled, cfg.KLV.Enabled); err != nil {
		return Config{}, err
	}
	if cfg.Weather.Enabled, err = boolFromEnv(getenv, EnvWeatherEnabled, cfg.Weather.Enabled); err != nil {
		return Config{}, err
	}
	if cfg.Fusion.Enabled, err = boolFromEnv(getenv, EnvFusionEnabled, cfg.Fusion.Enabled); err != nil {
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
	if cfg.CAP.HTTP.MaxResponseBytes, err = intFromEnv(
		getenv,
		EnvCAPHTTPMaxResponseBytes,
		cfg.CAP.HTTP.MaxResponseBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.ADSB.RawMaxRecords, err = intFromEnv(
		getenv,
		EnvADSBRawMaxRecords,
		cfg.ADSB.RawMaxRecords,
	); err != nil {
		return Config{}, err
	}
	if cfg.ADSB.RawMaxBytes, err = intFromEnv(
		getenv,
		EnvADSBRawMaxBytes,
		cfg.ADSB.RawMaxBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.ADSB.HTTP.MaxResponseBytes, err = intFromEnv(
		getenv,
		EnvADSBHTTPMaxResponseBytes,
		cfg.ADSB.HTTP.MaxResponseBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.SAPIENT.RawMaxRecords, err = intFromEnv(
		getenv,
		EnvSAPIENTRawMaxRecords,
		cfg.SAPIENT.RawMaxRecords,
	); err != nil {
		return Config{}, err
	}
	if cfg.SAPIENT.RawMaxBytes, err = intFromEnv(
		getenv,
		EnvSAPIENTRawMaxBytes,
		cfg.SAPIENT.RawMaxBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.SAPIENT.HTTP.MaxResponseBytes, err = intFromEnv(
		getenv,
		EnvSAPIENTHTTPMaxResponseBytes,
		cfg.SAPIENT.HTTP.MaxResponseBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.KLV.Demux.MaxPacketBytes, err = intFromEnv(
		getenv,
		EnvKLVDemuxMaxPacketBytes,
		cfg.KLV.Demux.MaxPacketBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.KLV.Demux.MaxExtractBytes, err = intFromEnv(
		getenv,
		EnvKLVDemuxMaxExtractBytes,
		cfg.KLV.Demux.MaxExtractBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.KLV.Demux.MaxPackets, err = intFromEnv(
		getenv,
		EnvKLVDemuxMaxPackets,
		cfg.KLV.Demux.MaxPackets,
	); err != nil {
		return Config{}, err
	}
	if cfg.KLV.Demux.MaxMaterializedBytes, err = intFromEnv(
		getenv,
		EnvKLVDemuxMaxMaterializedBytes,
		cfg.KLV.Demux.MaxMaterializedBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.KLV.Demux.ProbeOutputMaxBytes, err = intFromEnv(
		getenv,
		EnvKLVDemuxProbeOutputMaxBytes,
		cfg.KLV.Demux.ProbeOutputMaxBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.KLV.Decode.MaxPacketBytes, err = intFromEnv(
		getenv,
		EnvKLVDecodeMaxPacketBytes,
		cfg.KLV.Decode.MaxPacketBytes,
	); err != nil {
		return Config{}, err
	}
	if cfg.Weather.MaxObservations, err = intFromEnv(
		getenv,
		EnvWeatherMaxObservations,
		cfg.Weather.MaxObservations,
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
	if cfg.COP.MAVLinkSystemIDs, err = intListFromEnv(
		getenv,
		EnvCOPMAVLinkSystemIDs,
		cfg.COP.MAVLinkSystemIDs,
	); err != nil {
		return Config{}, err
	}
	if cfg.COP.CoTUIDs, err = stringListFromEnv(getenv, EnvCOPCoTUIDs, cfg.COP.CoTUIDs); err != nil {
		return Config{}, err
	}
	if cfg.COP.CAPAlertIDs, err = stringListFromEnv(
		getenv,
		EnvCOPCAPAlertIDs,
		cfg.COP.CAPAlertIDs,
	); err != nil {
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
	if !c.COP.GraphDiscoveryEnabled && len(c.COP.CoTUIDs) == 0 {
		return fmt.Errorf(
			"%s must include at least one UID when %s is false",
			EnvCOPCoTUIDs,
			EnvCOPGraphDiscoveryEnabled,
		)
	}
	if !c.COP.GraphDiscoveryEnabled && len(c.COP.CAPAlertIDs) == 0 {
		return fmt.Errorf(
			"%s must include at least one alert identifier when %s is false",
			EnvCOPCAPAlertIDs,
			EnvCOPGraphDiscoveryEnabled,
		)
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
	if err := c.CAP.Validate(); err != nil {
		return err
	}
	if err := c.ADSB.Validate(); err != nil {
		return err
	}
	if err := c.SAPIENT.Validate(); err != nil {
		return err
	}
	if err := c.KLV.Validate(); err != nil {
		return err
	}
	if err := c.Weather.Validate(); err != nil {
		return err
	}
	if err := c.Fusion.Validate(); err != nil {
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

func (c ADSBConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Source) == "" {
		return fmt.Errorf("%s is required when ADS-B is enabled", EnvADSBSource)
	}
	if strings.TrimSpace(c.Org) == "" {
		return fmt.Errorf("%s is required when ADS-B is enabled", EnvOrg)
	}
	if strings.TrimSpace(c.Platform) == "" {
		return fmt.Errorf("%s is required when ADS-B is enabled", EnvPlatform)
	}
	if c.RawMaxRecords <= 0 {
		return fmt.Errorf("%s must be greater than zero when ADS-B is enabled", EnvADSBRawMaxRecords)
	}
	if c.RawMaxBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when ADS-B is enabled", EnvADSBRawMaxBytes)
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("%s must be greater than zero when ADS-B is enabled", EnvADSBWriteTimeout)
	}
	if strings.TrimSpace(c.HTTP.URL) == "" {
		return fmt.Errorf("%s is required when ADS-B is enabled", EnvADSBHTTPURL)
	}
	if strings.TrimSpace(c.HTTP.Method) == "" {
		return fmt.Errorf("%s is required when ADS-B is enabled", EnvADSBHTTPMethod)
	}
	if c.HTTP.PollInterval <= 0 {
		return fmt.Errorf("%s must be greater than zero when ADS-B is enabled", EnvADSBHTTPPollInterval)
	}
	if c.HTTP.StaleAfter <= 0 {
		return fmt.Errorf("%s must be greater than zero when ADS-B is enabled", EnvADSBHTTPStaleAfter)
	}
	if c.HTTP.MaxResponseBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when ADS-B is enabled", EnvADSBHTTPMaxResponseBytes)
	}
	return nil
}

func (c SAPIENTConfig) Validate() error {
	if c.GraphEnabled && !c.Enabled {
		return fmt.Errorf("%s requires %s=true", EnvSAPIENTGraphEnabled, EnvSAPIENTEnabled)
	}
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Source) == "" {
		return fmt.Errorf("%s is required when SAPIENT is enabled", EnvSAPIENTSource)
	}
	if c.GraphEnabled {
		if strings.TrimSpace(c.Org) == "" {
			return fmt.Errorf("%s is required when SAPIENT graph projection is enabled", EnvOrg)
		}
		if strings.TrimSpace(c.Platform) == "" {
			return fmt.Errorf("%s is required when SAPIENT graph projection is enabled", EnvPlatform)
		}
		if strings.TrimSpace(c.TraceID) == "" {
			return fmt.Errorf("%s is required when SAPIENT graph projection is enabled", EnvTraceID)
		}
		if c.WriteTimeout <= 0 {
			return fmt.Errorf("%s must be greater than zero when SAPIENT graph projection is enabled", EnvSAPIENTWriteTimeout)
		}
	}
	if c.RawMaxRecords <= 0 {
		return fmt.Errorf("%s must be greater than zero when SAPIENT is enabled", EnvSAPIENTRawMaxRecords)
	}
	if c.RawMaxBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when SAPIENT is enabled", EnvSAPIENTRawMaxBytes)
	}
	if strings.TrimSpace(c.HTTP.URL) == "" {
		return fmt.Errorf("%s is required when SAPIENT is enabled", EnvSAPIENTHTTPURL)
	}
	if strings.TrimSpace(c.HTTP.Method) == "" {
		return fmt.Errorf("%s is required when SAPIENT is enabled", EnvSAPIENTHTTPMethod)
	}
	if c.HTTP.PollInterval <= 0 {
		return fmt.Errorf("%s must be greater than zero when SAPIENT is enabled", EnvSAPIENTHTTPPollInterval)
	}
	if c.HTTP.StaleAfter <= 0 {
		return fmt.Errorf("%s must be greater than zero when SAPIENT is enabled", EnvSAPIENTHTTPStaleAfter)
	}
	if c.HTTP.MaxResponseBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when SAPIENT is enabled", EnvSAPIENTHTTPMaxResponseBytes)
	}
	if c.HTTP.Encoding != "" && !c.HTTP.Encoding.Valid() {
		return fmt.Errorf("%s must be json, protobuf, or empty auto mode when SAPIENT is enabled", EnvSAPIENTHTTPEncoding)
	}
	return nil
}

func (c KLVConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Source) == "" {
		return fmt.Errorf("%s is required when KLV is enabled", EnvKLVSource)
	}
	if strings.TrimSpace(c.Org) == "" {
		return fmt.Errorf("%s is required when KLV is enabled", EnvOrg)
	}
	if strings.TrimSpace(c.Platform) == "" {
		return fmt.Errorf("%s is required when KLV is enabled", EnvPlatform)
	}
	if strings.TrimSpace(c.MediaPath) == "" {
		return fmt.Errorf("%s is required when KLV is enabled", EnvKLVMediaPath)
	}
	if strings.TrimSpace(c.MediaPattern) == "" {
		return fmt.Errorf("%s is required when KLV is enabled", EnvKLVMediaPattern)
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("%s must be greater than zero when KLV is enabled", EnvKLVWriteTimeout)
	}
	if c.Demux.MaxPacketBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when KLV is enabled", EnvKLVDemuxMaxPacketBytes)
	}
	if c.Demux.MaxExtractBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when KLV is enabled", EnvKLVDemuxMaxExtractBytes)
	}
	if c.Demux.MaxExtractBytes < c.Demux.MaxPacketBytes {
		return fmt.Errorf(
			"%s must be greater than or equal to %s when KLV is enabled",
			EnvKLVDemuxMaxExtractBytes,
			EnvKLVDemuxMaxPacketBytes,
		)
	}
	if c.Demux.MaxPackets <= 0 {
		return fmt.Errorf("%s must be greater than zero when KLV is enabled", EnvKLVDemuxMaxPackets)
	}
	if c.Demux.MaxMaterializedBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when KLV is enabled", EnvKLVDemuxMaxMaterializedBytes)
	}
	if c.Demux.ProbeOutputMaxBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when KLV is enabled", EnvKLVDemuxProbeOutputMaxBytes)
	}
	if c.Decode.MaxPacketBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when KLV is enabled", EnvKLVDecodeMaxPacketBytes)
	}
	return nil
}

func (c WeatherConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Source) == "" {
		return fmt.Errorf("%s is required when weather is enabled", EnvWeatherSource)
	}
	if strings.TrimSpace(c.Org) == "" {
		return fmt.Errorf("%s is required when weather is enabled", EnvOrg)
	}
	if strings.TrimSpace(c.Platform) == "" {
		return fmt.Errorf("%s is required when weather is enabled", EnvPlatform)
	}
	if strings.TrimSpace(c.Provider) == "" {
		return fmt.Errorf("%s is required when weather is enabled", EnvWeatherProvider)
	}
	if strings.TrimSpace(c.QueryShape) == "" {
		return fmt.Errorf("%s is required when weather is enabled", EnvWeatherQueryShape)
	}
	if strings.TrimSpace(c.FixturePath) == "" {
		return fmt.Errorf("%s is required when weather is enabled", EnvWeatherFixturePath)
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("%s must be greater than zero when weather is enabled", EnvWeatherWriteTimeout)
	}
	if c.Freshness <= 0 {
		return fmt.Errorf("%s must be greater than zero when weather is enabled", EnvWeatherFreshness)
	}
	if c.MaxObservations <= 0 {
		return fmt.Errorf("%s must be greater than zero when weather is enabled", EnvWeatherMaxObservations)
	}
	return nil
}

func (c FusionConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Org) == "" {
		return fmt.Errorf("%s is required when fusion is enabled", EnvOrg)
	}
	if strings.TrimSpace(c.Platform) == "" {
		return fmt.Errorf("%s is required when fusion is enabled", EnvPlatform)
	}
	if strings.TrimSpace(c.TraceID) == "" {
		return fmt.Errorf("%s is required when fusion is enabled", EnvTraceID)
	}
	if strings.TrimSpace(c.CandidateSubject) == "" {
		return fmt.Errorf("%s is required when fusion is enabled", EnvFusionCandidateSubject)
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("%s must be greater than zero when fusion is enabled", EnvFusionWriteTimeout)
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

func (c CAPConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Source) == "" {
		return fmt.Errorf("%s is required when CAP is enabled", EnvCAPSource)
	}
	if strings.TrimSpace(c.Org) == "" {
		return fmt.Errorf("%s is required when CAP is enabled", EnvOrg)
	}
	if strings.TrimSpace(c.Platform) == "" {
		return fmt.Errorf("%s is required when CAP is enabled", EnvPlatform)
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("%s must be greater than zero when CAP is enabled", EnvCAPWriteTimeout)
	}
	if strings.TrimSpace(c.HTTP.URL) == "" {
		return fmt.Errorf("%s is required when CAP is enabled", EnvCAPHTTPURL)
	}
	if strings.TrimSpace(c.HTTP.Method) == "" {
		return fmt.Errorf("%s is required when CAP is enabled", EnvCAPHTTPMethod)
	}
	if c.HTTP.PollInterval <= 0 {
		return fmt.Errorf("%s must be greater than zero when CAP is enabled", EnvCAPHTTPPollInterval)
	}
	if c.HTTP.StaleAfter <= 0 {
		return fmt.Errorf("%s must be greater than zero when CAP is enabled", EnvCAPHTTPStaleAfter)
	}
	if c.HTTP.MaxResponseBytes <= 0 {
		return fmt.Errorf("%s must be greater than zero when CAP is enabled", EnvCAPHTTPMaxResponseBytes)
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
