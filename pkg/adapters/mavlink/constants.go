package mavlink

const (
	Version1 uint8 = 1
	Version2 uint8 = 2

	STXV1 byte = 0xfe
	STXV2 byte = 0xfd

	MaxPayloadLength = 255
	HeaderSizeV1     = 6
	HeaderSizeV2     = 10
	ChecksumSize     = 2
	SignatureSize    = 13
)

const (
	MessageIDHeartbeat         uint32 = 0
	MessageIDAttitude          uint32 = 30
	MessageIDGlobalPositionInt uint32 = 33
	MessageIDCommandLong       uint32 = 76
	MessageIDCommandAck        uint32 = 77
	MessageIDBatteryStatus     uint32 = 147
)

const (
	TypeGeneric   uint8 = 0
	TypeQuadrotor uint8 = 2
	TypeGCS       uint8 = 6
)

const (
	AutopilotGeneric         uint8 = 0
	AutopilotArduPilotMega   uint8 = 3
	AutopilotPX4             uint8 = 12
	AutopilotInvalid         uint8 = 8
	ModeFlagStabilizeEnabled uint8 = 16
	ModeFlagManualInput      uint8 = 64
	ModeFlagSafetyArmed      uint8 = 128
)

const (
	StateUninit  uint8 = 0
	StateStandby uint8 = 3
	StateActive  uint8 = 4
)

const (
	CommandNavWaypoint            uint16 = 16
	CommandNavReturnToLaunch      uint16 = 20
	CommandNavLand                uint16 = 21
	CommandNavTakeoff             uint16 = 22
	CommandConditionYaw           uint16 = 115
	CommandDoSetMode              uint16 = 176
	CommandComponentArmDisarm     uint16 = 400
	CommandSetMessageInterval     uint16 = 511
	CommandRequestAutopilotCaps   uint16 = 520
	CommandVideoStartCapture      uint16 = 2500
	CommandVideoStopCapture       uint16 = 2501
	CommandVideoStartStreaming    uint16 = 2502
	CommandVideoStopStreaming     uint16 = 2503
	CommandRequestVideoStreamInfo uint16 = 2504
)

const (
	MAVResultAccepted            uint8 = 0
	MAVResultTemporarilyRejected uint8 = 1
	MAVResultDenied              uint8 = 2
	MAVResultUnsupported         uint8 = 3
	MAVResultFailed              uint8 = 4
	MAVResultInProgress          uint8 = 5
	MAVResultCancelled           uint8 = 6
)
