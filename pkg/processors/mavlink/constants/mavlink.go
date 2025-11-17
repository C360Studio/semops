// Package constants provides MAVLink protocol constants for robotics domain.
// All constants follow Go naming conventions (CamelCase) while maintaining
// compatibility with MAVLink 2.0 specification for autonomous vehicle communication.
package constants

const (
	// MAVLink Protocol Versions
	
	// MavlinkV1 represents MAVLink protocol version 1
	MavlinkV1 = 1
	// MavlinkV2 represents MAVLink protocol version 2
	MavlinkV2 = 2
	
	// MAVLink packet format constants
	
	// MavlinkStxV1 is the sync byte for MAVLink version 1 packets
	MavlinkStxV1 = 0xFE
	// MavlinkStxV2 is the sync byte for MAVLink version 2 packets
	MavlinkStxV2 = 0xFD
	// MavlinkMaxPayloadLen is the maximum payload length in MAVLink packets
	MavlinkMaxPayloadLen = 255
	// MavlinkHeaderSizeV1 is the header size for MAVLink version 1
	MavlinkHeaderSizeV1 = 6
	// MavlinkHeaderSizeV2 is the header size for MAVLink version 2
	MavlinkHeaderSizeV2 = 10
	// MavlinkChecksumSize is the checksum size in MAVLink packets
	MavlinkChecksumSize = 2
	// MavlinkSignatureSize is the signature size for MAVLink version 2
	MavlinkSignatureSize = 13
)

// Vehicle Types (MAV_TYPE)
const (
	// MavTypeGeneric represents a generic micro air vehicle
	MavTypeGeneric = 0
	// MavTypeFixedWing represents a fixed wing aircraft
	MavTypeFixedWing = 1
	// MavTypeQuadrotor represents a quadrotor
	MavTypeQuadrotor = 2
	// MavTypeCoaxial represents a coaxial helicopter
	MavTypeCoaxial = 3
	// MavTypeHelicopter represents a normal helicopter with tail rotor
	MavTypeHelicopter = 4
	// MavTypeAntennaTracker represents a ground installation
	MavTypeAntennaTracker = 5
	// MavTypeGcs represents an operator control unit / ground control station
	MavTypeGcs = 6
	// MavTypeAirship represents an airship, controlled
	MavTypeAirship = 7
	// MavTypeFreeBalloon represents a free balloon, uncontrolled
	MavTypeFreeBalloon = 8
	// MavTypeRocket represents a rocket
	MavTypeRocket = 9
	// MavTypeGroundRover represents a ground rover
	MavTypeGroundRover = 10
	// MavTypeSurfaceBoat represents a surface vessel, boat, ship
	MavTypeSurfaceBoat = 11
	// MavTypeSubmarine represents a submarine
	MavTypeSubmarine = 12
	// MavTypeHexarotor represents a hexarotor
	MavTypeHexarotor = 13
	// MavTypeOctorotor represents an octorotor
	MavTypeOctorotor = 14
	// MavTypeTricopter represents a tricopter
	MavTypeTricopter = 15
	// MavTypeFlappingWing represents a flapping wing
	MavTypeFlappingWing = 16
	// MavTypeKite represents a kite
	MavTypeKite = 17
	// MavTypeOnboardController represents an onboard companion controller
	MavTypeOnboardController = 18
	// MavTypeVtolDuorotor represents a two-rotor VTOL using control surfaces in vertical operation
	MavTypeVtolDuorotor = 19
	// MavTypeVtolQuadrotor represents a quad-rotor VTOL using a V-tail quad config
	MavTypeVtolQuadrotor = 20
	// MavTypeVtolTiltrotor represents a tiltrotor VTOL
	MavTypeVtolTiltrotor = 21
	// MavTypeVtolReserved2 represents VTOL reserved 2
	MavTypeVtolReserved2 = 22
	// MavTypeVtolReserved3 represents VTOL reserved 3
	MavTypeVtolReserved3 = 23
	// MavTypeVtolReserved4 represents VTOL reserved 4
	MavTypeVtolReserved4 = 24
	// MavTypeVtolReserved5 represents VTOL reserved 5
	MavTypeVtolReserved5 = 25
	// MavTypeGimbal represents a gimbal
	MavTypeGimbal = 26
	// MavTypeAdsb represents an ADSB system
	MavTypeAdsb = 27
	// MavTypeParafoil represents a steerable, nonrigid airfoil
	MavTypeParafoil = 28
	// MavTypeDodecarotor represents a dodecarotor
	MavTypeDodecarotor = 29
	// MavTypeCamera represents a camera
	MavTypeCamera = 30
	// MavTypeChargingStation represents a charging station
	MavTypeChargingStation = 31
	// MavTypeFlarm represents a FLARM collision avoidance system
	MavTypeFlarm = 32
	// MavTypeServo represents a servo
	MavTypeServo = 33
)

// Autopilot Types (MAV_AUTOPILOT)
const (
	// MavAutopilotGeneric represents a generic autopilot, full support for everything
	MavAutopilotGeneric = 0
	// MavAutopilotReserved is reserved for future use
	MavAutopilotReserved = 1
	// MavAutopilotSlugs represents a SLUGS autopilot
	MavAutopilotSlugs = 2
	// MavAutopilotArdupilotmega represents ArduPilot - Plane/Copter/Rover/Sub/Tracker
	MavAutopilotArdupilotmega = 3
	// MavAutopilotOpenpilot represents OpenPilot
	MavAutopilotOpenpilot = 4
	// MavAutopilotGenericWaypointsOnly represents a generic autopilot only supporting simple waypoints
	MavAutopilotGenericWaypointsOnly = 5
	// MavAutopilotGenericWaypointsAndSimpleNavigationOnly represents a generic autopilot supporting waypoints and other simple navigation commands
	MavAutopilotGenericWaypointsAndSimpleNavigationOnly = 6
	// MavAutopilotGenericMissionFull represents a generic autopilot supporting the full mission command set
	MavAutopilotGenericMissionFull = 7
	// MavAutopilotInvalid represents no valid autopilot, e.g. a GCS or other MAVLink component
	MavAutopilotInvalid = 8
	// MavAutopilotPpz represents a PPZ UAV
	MavAutopilotPpz = 9
	// MavAutopilotUdb represents a UAV Dev Board
	MavAutopilotUdb = 10
	// MavAutopilotFp represents a FlexiPilot
	MavAutopilotFp = 11
	// MavAutopilotPx4 represents a PX4 Autopilot
	MavAutopilotPx4 = 12
	// MavAutopilotSmaccmpilot represents a SMACCMPilot
	MavAutopilotSmaccmpilot = 13
	// MavAutopilotAutoquad represents an AutoQuad
	MavAutopilotAutoquad = 14
	// MavAutopilotArmazila represents an Armazila
	MavAutopilotArmazila = 15
	// MavAutopilotAerob represents an Aerob
	MavAutopilotAerob = 16
	// MavAutopilotAsluav represents an ASLUAV autopilot
	MavAutopilotAsluav = 17
	// MavAutopilotSmartap represents a SmartAP Autopilot
	MavAutopilotSmartap = 18
	// MavAutopilotAirrails represents AirRails
	MavAutopilotAirrails = 19
)

// System Status (MAV_STATE)
const (
	// MavStateUninit represents an uninitialized system, LEDs show heartbeat
	MavStateUninit = 0
	// MavStateBoot represents a system that is booting up
	MavStateBoot = 1
	// MavStateCalibrating represents a system that is calibrating and not flight-ready
	MavStateCalibrating = 2
	// MavStateStandby represents a system that is grounded and ready for flight
	MavStateStandby = 3
	// MavStateActive represents a system that is active and might be already airborne
	MavStateActive = 4
	// MavStateCritical represents a system that is in a non-normal flight mode
	MavStateCritical = 5
	// MavStateEmergency represents a system that is in an emergency state
	MavStateEmergency = 6
	// MavStatePoweroff represents a system that is about to shut down
	MavStatePoweroff = 7
	// MavStateFlightTermination represents a system that is terminating the flight
	MavStateFlightTermination = 8
)

// Base Mode Flags (MAV_MODE_FLAG)
const (
	// MavModeFlagCustomModeEnabled is reserved for future use
	MavModeFlagCustomModeEnabled = 1
	// MavModeFlagTestEnabled indicates system has a test mode enabled
	MavModeFlagTestEnabled = 2
	// MavModeFlagAutoEnabled indicates autonomous mode enabled, system finds its own goal positions
	MavModeFlagAutoEnabled = 4
	// MavModeFlagGuidedEnabled indicates guided mode enabled, system flies waypoints / mission items
	MavModeFlagGuidedEnabled = 8
	// MavModeFlagStabilizeEnabled indicates stabilize mode enabled, system stabilizes electronically its attitude
	MavModeFlagStabilizeEnabled = 16
	// MavModeFlagHilEnabled indicates hardware in the loop simulation
	MavModeFlagHilEnabled = 32
	// MavModeFlagManualInputEnabled indicates remote control input is enabled
	MavModeFlagManualInputEnabled = 64
	// MavModeFlagSafetyArmed indicates system safety flag, motors are enabled / running / can start
	MavModeFlagSafetyArmed = 128
)

// MAVLink Message IDs
const (
	// MavlinkMsgIdHeartbeat represents the heartbeat message shows that a system is present
	MavlinkMsgIdHeartbeat = 0
	// MavlinkMsgIdSysStatus represents system status including onboard control sensors
	MavlinkMsgIdSysStatus = 1
	// MavlinkMsgIdSystemTime represents system time and GPS time
	MavlinkMsgIdSystemTime = 2
	// MavlinkMsgIdPing represents a ping message to initiate a link
	MavlinkMsgIdPing = 4
	// MavlinkMsgIdChangeOperatorControl represents a request to control this MAV
	MavlinkMsgIdChangeOperatorControl = 5
	// MavlinkMsgIdChangeOperatorControlAck represents accept / deny control of this MAV
	MavlinkMsgIdChangeOperatorControlAck = 6
	// MavlinkMsgIdAuthKey represents an authentication key
	MavlinkMsgIdAuthKey = 7
	// MavlinkMsgIdSetMode represents set the system mode
	MavlinkMsgIdSetMode = 11
	// MavlinkMsgIdParamRequestRead represents request to read the onboard parameter with the param_id string id
	MavlinkMsgIdParamRequestRead = 20
	// MavlinkMsgIdParamRequestList represents request all parameters of this component
	MavlinkMsgIdParamRequestList = 21
	// MavlinkMsgIdParamValue represents parameter value from onboard control system
	MavlinkMsgIdParamValue = 22
	// MavlinkMsgIdParamSet represents set a parameter value TEMPORARILY
	MavlinkMsgIdParamSet = 23
	// MavlinkMsgIdGpsRawInt represents GPS raw sensor values
	MavlinkMsgIdGpsRawInt = 24
	// MavlinkMsgIdGpsStatus represents GPS status
	MavlinkMsgIdGpsStatus = 25
	// MavlinkMsgIdScaledImu represents raw IMU readings for a 9DOF sensor
	MavlinkMsgIdScaledImu = 26
	// MavlinkMsgIdRawImu represents raw IMU readings for a 9DOF sensor
	MavlinkMsgIdRawImu = 27
	// MavlinkMsgIdRawPressure represents barometer readings
	MavlinkMsgIdRawPressure = 28
	// MavlinkMsgIdScaledPressure represents barometer readings with pressure altitude
	MavlinkMsgIdScaledPressure = 29
	// MavlinkMsgIdAttitude represents attitude (roll, pitch, yaw) and angular rates
	MavlinkMsgIdAttitude = 30
	// MavlinkMsgIdAttitudeQuaternion represents attitude quaternion and angular rates
	MavlinkMsgIdAttitudeQuaternion = 31
	// MavlinkMsgIdLocalPositionNed represents position in local coordinate frame
	MavlinkMsgIdLocalPositionNed = 32
	// MavlinkMsgIdGlobalPositionInt represents position in WGS84 coordinate frame
	MavlinkMsgIdGlobalPositionInt = 33
	// MavlinkMsgIdRcChannelsScaled represents RC channels scaled to [-1000,1000] range
	MavlinkMsgIdRcChannelsScaled = 34
	// MavlinkMsgIdRcChannelsRaw represents RC channels raw PWM values
	MavlinkMsgIdRcChannelsRaw = 35
	// MavlinkMsgIdServoOutputRaw represents servo output raw PWM values
	MavlinkMsgIdServoOutputRaw = 36
	// MavlinkMsgIdMissionRequestPartialList represents request partial list of mission items
	MavlinkMsgIdMissionRequestPartialList = 37
	// MavlinkMsgIdMissionWritePartialList represents request to write a partial list of mission items
	MavlinkMsgIdMissionWritePartialList = 38
	// MavlinkMsgIdMissionItem represents message encoding a mission item
	MavlinkMsgIdMissionItem = 39
	// MavlinkMsgIdMissionRequest represents request the information of the mission item with the sequence number seq
	MavlinkMsgIdMissionRequest = 40
	// MavlinkMsgIdMissionSetCurrent represents set the mission item with sequence number seq as current item
	MavlinkMsgIdMissionSetCurrent = 41
	// MavlinkMsgIdMissionCurrent represents current mission item sequence number
	MavlinkMsgIdMissionCurrent = 42
	// MavlinkMsgIdMissionRequestList represents request the overall list of mission items from the system/component
	MavlinkMsgIdMissionRequestList = 43
	// MavlinkMsgIdMissionCount represents number of mission items on the system
	MavlinkMsgIdMissionCount = 44
	// MavlinkMsgIdMissionClearAll represents delete all mission items at once
	MavlinkMsgIdMissionClearAll = 45
	// MavlinkMsgIdMissionItemReached represents a certain mission item has been reached
	MavlinkMsgIdMissionItemReached = 46
	// MavlinkMsgIdMissionAck represents acknowledgement message during waypoint handling
	MavlinkMsgIdMissionAck = 47
	// MavlinkMsgIdSetGpsGlobalOrigin represents set GPS global origin
	MavlinkMsgIdSetGpsGlobalOrigin = 48
	// MavlinkMsgIdGpsGlobalOrigin represents GPS global origin
	MavlinkMsgIdGpsGlobalOrigin = 49
	// MavlinkMsgIdParamMapRc represents bind a RC channel to a parameter
	MavlinkMsgIdParamMapRc = 50
	// MavlinkMsgIdMissionRequestInt represents request the information of the mission item with the sequence number seq
	MavlinkMsgIdMissionRequestInt = 51
	// MavlinkMsgIdSafetySetAllowedArea represents set safety allowed area
	MavlinkMsgIdSafetySetAllowedArea = 54
	// MavlinkMsgIdSafetyAllowedArea represents read safety allowed area
	MavlinkMsgIdSafetyAllowedArea = 55
	// MavlinkMsgIdAttitudeQuaternionCov represents attitude covariance matrix and angular rates
	MavlinkMsgIdAttitudeQuaternionCov = 61
	// MavlinkMsgIdNavControllerOutput represents navigation controller output
	MavlinkMsgIdNavControllerOutput = 62
	// MavlinkMsgIdGlobalPositionIntCov represents position and velocity in WGS84 coordinate frame with covariance
	MavlinkMsgIdGlobalPositionIntCov = 63
	// MavlinkMsgIdLocalPositionNedCov represents position and velocity in local coordinate frame with covariance
	MavlinkMsgIdLocalPositionNedCov = 64
	// MavlinkMsgIdRcChannels represents RC channels
	MavlinkMsgIdRcChannels = 65
	// MavlinkMsgIdRequestDataStream represents request a data stream
	MavlinkMsgIdRequestDataStream = 66
	// MavlinkMsgIdDataStream represents data stream status information
	MavlinkMsgIdDataStream = 67
	// MavlinkMsgIdManualControl represents manual control input from operator
	MavlinkMsgIdManualControl = 69
	// MavlinkMsgIdRcChannelsOverride represents RC channel override
	MavlinkMsgIdRcChannelsOverride = 70
	// MavlinkMsgIdMissionItemInt represents message encoding a mission item with sequence number and frame
	MavlinkMsgIdMissionItemInt = 73
	// MavlinkMsgIdVfrHud represents metrics typically displayed on HUD
	MavlinkMsgIdVfrHud = 74
	// MavlinkMsgIdCommandInt represents message encoding a command with parameters as scaled integers
	MavlinkMsgIdCommandInt = 75
	// MavlinkMsgIdCommandLong represents send a command with up to seven parameters
	MavlinkMsgIdCommandLong = 76
	// MavlinkMsgIdCommandAck represents report status of a command
	MavlinkMsgIdCommandAck = 77
	// MavlinkMsgIdBatteryStatus represents battery information
	MavlinkMsgIdBatteryStatus = 147
	MAVLINK_MSG_ID_MANUAL_SETPOINT        = 81  // Manual setpoint for a control surface
	MAVLINK_MSG_ID_SET_ATTITUDE_TARGET    = 82  // Sets a desired vehicle attitude
	MAVLINK_MSG_ID_ATTITUDE_TARGET        = 83  // Reports the current commanded attitude of the vehicle
	MAVLINK_MSG_ID_SET_POSITION_TARGET_LOCAL_NED = 84 // Sets a desired vehicle position in a local north-east-down coordinate frame
	MAVLINK_MSG_ID_POSITION_TARGET_LOCAL_NED = 85     // Reports the current commanded vehicle position, velocity, and acceleration
	MAVLINK_MSG_ID_SET_POSITION_TARGET_GLOBAL_INT = 86 // Sets a desired vehicle position and/or attitude in the WGS84 coordinate system
	MAVLINK_MSG_ID_POSITION_TARGET_GLOBAL_INT = 87     // Reports the current commanded vehicle position, velocity, and acceleration
	MAVLINK_MSG_ID_LOCAL_POSITION_NED_SYSTEM_GLOBAL_OFFSET = 89 // Local position measurement in a system global frame
	MAVLINK_MSG_ID_HIL_STATE              = 90  // Hardware in the Loop state
	MAVLINK_MSG_ID_HIL_CONTROLS           = 91  // Hardware in the Loop control interface
	MAVLINK_MSG_ID_HIL_RC_INPUTS_RAW      = 92  // Hardware in the Loop RC channel inputs
	MAVLINK_MSG_ID_HIL_ACTUATOR_CONTROLS  = 93  // Hardware in the Loop actuator controls
	MAVLINK_MSG_ID_OPTICAL_FLOW           = 100 // Optical flow from sensor
	MAVLINK_MSG_ID_GLOBAL_VISION_POSITION_ESTIMATE = 101 // Global vision position estimate
	MAVLINK_MSG_ID_VISION_POSITION_ESTIMATE = 102        // Local vision position estimate
	MAVLINK_MSG_ID_VISION_SPEED_ESTIMATE  = 103          // Speed estimate from vision sensor
	MAVLINK_MSG_ID_VICON_POSITION_ESTIMATE = 104         // Global position estimate from a Vicon motion capture system
	MAVLINK_MSG_ID_HIGHRES_IMU            = 105          // High resolution IMU data
	MAVLINK_MSG_ID_OPTICAL_FLOW_RAD       = 106          // Optical flow from sensor, in radians per second angular rate
	MAVLINK_MSG_ID_HIL_SENSOR             = 107          // Sensor readings for Hardware in the Loop
	MAVLINK_MSG_ID_SIM_STATE              = 108          // Simulation state
	MAVLINK_MSG_ID_RADIO_STATUS           = 109          // Status of radio link
	MAVLINK_MSG_ID_FILE_TRANSFER_PROTOCOL = 110          // File transfer protocol message
	MAVLINK_MSG_ID_TIMESYNC               = 111          // Time synchronization message
	MAVLINK_MSG_ID_CAMERA_TRIGGER         = 112          // Camera trigger message
	MAVLINK_MSG_ID_HIL_GPS                = 113          // GPS data for Hardware in the Loop
	MAVLINK_MSG_ID_HIL_OPTICAL_FLOW       = 114          // Optical flow from sensor for Hardware in the Loop
	MAVLINK_MSG_ID_HIL_STATE_QUATERNION   = 115          // Hardware in the Loop state using quaternion attitude representation
	MAVLINK_MSG_ID_SCALED_IMU2            = 116          // Second set of IMU readings
	MAVLINK_MSG_ID_LOG_REQUEST_LIST       = 117          // Request list of log files
	MAVLINK_MSG_ID_LOG_ENTRY              = 118          // Reply to LOG_REQUEST_LIST
	MAVLINK_MSG_ID_LOG_REQUEST_DATA       = 119          // Request log data
	MAVLINK_MSG_ID_LOG_DATA               = 120          // Reply to LOG_REQUEST_DATA
	MAVLINK_MSG_ID_LOG_ERASE              = 121          // Erase all logs
	MAVLINK_MSG_ID_LOG_REQUEST_END        = 122          // Stop log data transfer
	MAVLINK_MSG_ID_GPS_INJECT_DATA        = 123          // GPS RTK correction data
	MAVLINK_MSG_ID_GPS2_RAW               = 124          // Second GPS data set
	MAVLINK_MSG_ID_POWER_STATUS           = 125          // Power supply status
	MAVLINK_MSG_ID_SERIAL_CONTROL         = 126          // Control a serial port
	MAVLINK_MSG_ID_GPS_RTK                = 127          // GPS RTK data
	MAVLINK_MSG_ID_GPS2_RTK               = 128          // Second set of GPS RTK data
	MAVLINK_MSG_ID_SCALED_IMU3            = 129          // Third set of IMU readings
	MAVLINK_MSG_ID_DATA_TRANSMISSION_HANDSHAKE = 130     // Handshake message to initiate, control and stop image streaming
	MAVLINK_MSG_ID_ENCAPSULATED_DATA      = 131          // Encapsulate data for transmission
	MAVLINK_MSG_ID_DISTANCE_SENSOR        = 132          // Distance sensor information
	MAVLINK_MSG_ID_TERRAIN_REQUEST        = 133          // Request terrain data
	MAVLINK_MSG_ID_TERRAIN_DATA           = 134          // Terrain data sent from GCS
	MAVLINK_MSG_ID_TERRAIN_CHECK          = 135          // Request that terrain data be loaded
	MAVLINK_MSG_ID_TERRAIN_REPORT         = 136          // Terrain status report
	MAVLINK_MSG_ID_SCALED_PRESSURE2       = 137          // Second set of barometer readings
	MAVLINK_MSG_ID_ATT_POS_MOCAP          = 138          // Motion capture attitude and position
	MAVLINK_MSG_ID_SET_ACTUATOR_CONTROL_TARGET = 139     // Set actuator targets
	MAVLINK_MSG_ID_ACTUATOR_CONTROL_TARGET = 140         // Current actuator target values
	MAVLINK_MSG_ID_ALTITUDE               = 141          // Altitude readings
	MAVLINK_MSG_ID_RESOURCE_REQUEST       = 142          // Request for a resource
	MAVLINK_MSG_ID_SCALED_PRESSURE3       = 143          // Third set of barometer readings
	MAVLINK_MSG_ID_FOLLOW_TARGET          = 144          // Target for camera tracking
	MAVLINK_MSG_ID_CONTROL_SYSTEM_STATE   = 146          // System control state
	MAVLINK_MSG_ID_BATTERY_STATUS         = 147          // Battery information
	MAVLINK_MSG_ID_AUTOPILOT_VERSION      = 148          // Autopilot version and capability
	MAVLINK_MSG_ID_LANDING_TARGET         = 149          // Landing target position
	MAVLINK_MSG_ID_FENCE_STATUS           = 162          // Geofence status
	MAVLINK_MSG_ID_ESTIMATOR_STATUS       = 230          // Estimator status message including flags, innovation test ratios and estimated accuracies
	MAVLINK_MSG_ID_WIND_COV               = 231          // Wind covariance estimate from vehicle
	MAVLINK_MSG_ID_GPS_INPUT              = 232          // GPS sensor input message
	MAVLINK_MSG_ID_GPS_RTCM_DATA          = 233          // RTCM message for injecting into the onboard GPS
	MAVLINK_MSG_ID_HIGH_LATENCY           = 234          // Message appropriate for high latency connections
	MAVLINK_MSG_ID_HIGH_LATENCY2          = 235          // Message appropriate for high latency connections
	MAVLINK_MSG_ID_VIBRATION              = 241          // Vibration levels and accelerometer clipping
	MAVLINK_MSG_ID_HOME_POSITION          = 242          // Home position from GPS
	MAVLINK_MSG_ID_SET_HOME_POSITION      = 243          // Set home position from GPS coordinates
	MAVLINK_MSG_ID_MESSAGE_INTERVAL       = 244          // Configure message interval
	MAVLINK_MSG_ID_EXTENDED_SYS_STATE     = 245          // Extended system state
	MAVLINK_MSG_ID_ADSB_VEHICLE           = 246          // ADS-B vehicle information
	MAVLINK_MSG_ID_COLLISION              = 247          // Collision avoidance system information
	MAVLINK_MSG_ID_V2_EXTENSION           = 248          // Message implementing parts of the V2 payload specs in V1 frames
	MAVLINK_MSG_ID_MEMORY_VECT            = 249          // Memory vector message
	MAVLINK_MSG_ID_DEBUG_VECT             = 250          // Debug vector message
	MAVLINK_MSG_ID_NAMED_VALUE_FLOAT      = 251          // Named float value
	MAVLINK_MSG_ID_NAMED_VALUE_INT        = 252          // Named integer value
	MAVLINK_MSG_ID_STATUSTEXT             = 253          // Status text message
	MAVLINK_MSG_ID_DEBUG                  = 254          // Debug message
	MAVLINK_MSG_ID_SETUP_SIGNING          = 256          // Setup packet signing
)

// Mission Command IDs (MAV_CMD)
const (
	MAV_CMD_NAV_WAYPOINT                      = 16   // Navigate to waypoint
	MAV_CMD_NAV_LOITER_UNLIM                  = 17   // Loiter around this waypoint an unlimited amount of time
	MAV_CMD_NAV_LOITER_TURNS                  = 18   // Loiter around this waypoint for X turns
	MAV_CMD_NAV_LOITER_TIME                   = 19   // Loiter around this waypoint for X seconds
	MAV_CMD_NAV_RETURN_TO_LAUNCH              = 20   // Return to launch location
	MAV_CMD_NAV_LAND                          = 21   // Land at location
	MAV_CMD_NAV_TAKEOFF                       = 22   // Takeoff from ground / hand
	MAV_CMD_NAV_LAND_LOCAL                    = 23   // Land at local position (local frame only)
	MAV_CMD_NAV_TAKEOFF_LOCAL                 = 24   // Takeoff from local position (local frame only)
	MAV_CMD_NAV_FOLLOW                        = 25   // Vehicle following, i.e. this waypoint represents the position of a moving vehicle
	MAV_CMD_NAV_CONTINUE_AND_CHANGE_ALT       = 30   // Continue on the current course and climb/descend to specified altitude
	MAV_CMD_NAV_LOITER_TO_ALT                 = 31   // Begin loiter at the specified Latitude and Longitude
	MAV_CMD_DO_FOLLOW                         = 32   // Begin following a target
	MAV_CMD_DO_FOLLOW_REPOSITION              = 33   // Reposition the MAV after a follow target command has been sent
	MAV_CMD_NAV_ROI                           = 80   // Sets the region of interest (ROI) for a sensor set or the vehicle itself
	MAV_CMD_NAV_PATHPLANNING                  = 81   // Control autonomous path planning on the MAV
	MAV_CMD_NAV_SPLINE_WAYPOINT               = 82   // Navigate to waypoint using a spline path
	MAV_CMD_NAV_VTOL_TAKEOFF                  = 84   // Takeoff from ground using VTOL mode
	MAV_CMD_NAV_VTOL_LAND                     = 85   // Land using VTOL mode
	MAV_CMD_NAV_GUIDED_ENABLE                 = 92   // Enable/disable guided mode
	MAV_CMD_NAV_DELAY                         = 93   // Delay the next navigation command a number of seconds or until a specified time
	MAV_CMD_NAV_PAYLOAD_PLACE                 = 94   // Descend and place payload
	MAV_CMD_NAV_LAST                          = 95   // Last waypoint marker
	MAV_CMD_CONDITION_DELAY                   = 112  // Delay mission state machine
	MAV_CMD_CONDITION_CHANGE_ALT              = 113  // Ascend/descend at rate
	MAV_CMD_CONDITION_DISTANCE                = 114  // Delay mission state machine until within desired distance of next NAV point
	MAV_CMD_CONDITION_YAW                     = 115  // Reach a certain target angle
	MAV_CMD_CONDITION_LAST                    = 159  // Last condition marker
	MAV_CMD_DO_SET_MODE                       = 176  // Set system mode
	MAV_CMD_DO_JUMP                           = 177  // Jump to the desired command in the mission list
	MAV_CMD_DO_CHANGE_SPEED                   = 178  // Change speed and/or throttle set points
	MAV_CMD_DO_SET_HOME                       = 179  // Changes the home location either to the current location or a specified location
	MAV_CMD_DO_SET_PARAMETER                  = 180  // Set a system parameter
	MAV_CMD_DO_SET_RELAY                      = 181  // Set a relay to a condition
	MAV_CMD_DO_REPEAT_RELAY                   = 182  // Cycle a relay on and off for a desired number of cycles
	MAV_CMD_DO_SET_SERVO                      = 183  // Set a servo to a desired PWM value
	MAV_CMD_DO_REPEAT_SERVO                   = 184  // Cycle a between its nominal setting and a desired PWM for a desired number of cycles
	MAV_CMD_DO_FLIGHTTERMINATION              = 185  // Terminate flight immediately
	MAV_CMD_DO_CHANGE_ALTITUDE                = 186  // Change altitude set point
	MAV_CMD_DO_LAND_START                     = 189  // Mission command to perform a landing
	MAV_CMD_DO_RALLY_LAND                     = 190  // Mission command to safely abort an autonomous landing
	MAV_CMD_DO_GO_AROUND                      = 191  // Mission command to perform a go around
	MAV_CMD_DO_REPOSITION                     = 192  // Reposition the vehicle to a specific WGS84 global position
	MAV_CMD_DO_PAUSE_CONTINUE                 = 193  // Pause or continue the mission
	MAV_CMD_DO_SET_REVERSE                    = 194  // Set moving direction to forward or reverse
	MAV_CMD_DO_SET_ROI_LOCATION               = 195  // Sets the region of interest (ROI) to a location
	MAV_CMD_DO_SET_ROI_WPNEXT_OFFSET          = 196  // Sets the region of interest (ROI) to be toward next waypoint
	MAV_CMD_DO_SET_ROI_NONE                   = 197  // Cancels any previous ROI command returning the vehicle/sensors to default behavior
	MAV_CMD_DO_CONTROL_VIDEO                  = 200  // Control onboard camera system
	MAV_CMD_DO_SET_CAM_TRIGG_DIST             = 206  // Camera trigger distance
	MAV_CMD_DO_FENCE_ENABLE                   = 207  // Enable geofence
	MAV_CMD_DO_PARACHUTE                      = 208  // Trigger parachute
	MAV_CMD_DO_MOTOR_TEST                     = 209  // Motor test
	MAV_CMD_DO_INVERTED_FLIGHT                = 210  // Change to/from inverted flight
	MAV_CMD_NAV_SET_YAW_SPEED                 = 213  // Set the yaw and ground speed
	MAV_CMD_DO_SET_CAM_TRIGG_INTERVAL         = 214  // Camera trigger interval
	MAV_CMD_DO_MOUNT_CONTROL_QUAT             = 220  // Mission command to control a camera or antenna mount using a quaternion as reference
	MAV_CMD_DO_GUIDED_MASTER                  = 221  // Set MAV as guided master
	MAV_CMD_DO_GUIDED_LIMITS                  = 222  // Set limits for external guided mode
	MAV_CMD_DO_ENGINE_CONTROL                 = 223  // Control vehicle engine
	MAV_CMD_DO_SET_MISSION_CURRENT            = 224  // Set current mission index
	MAV_CMD_DO_LAST                           = 240  // Last do command marker
	MAV_CMD_PREFLIGHT_CALIBRATION             = 241  // Trigger calibration
	MAV_CMD_PREFLIGHT_SET_SENSOR_OFFSETS      = 242  // Set sensor offsets
	MAV_CMD_PREFLIGHT_UAVCAN                  = 243  // Trigger UAVCAN config
	MAV_CMD_PREFLIGHT_STORAGE                 = 245  // Request storage of different parameter values and logs
	MAV_CMD_PREFLIGHT_REBOOT_SHUTDOWN         = 246  // Request the reboot or shutdown of system components
	MAV_CMD_OVERRIDE_GOTO                     = 252  // Override previous command
	MAV_CMD_MISSION_START                     = 300  // Start running a mission
	MAV_CMD_COMPONENT_ARM_DISARM              = 400  // Arms / Disarms a component
	MAV_CMD_GET_HOME_POSITION                 = 410  // Request the home position from the vehicle
	MAV_CMD_START_RX_PAIR                     = 500  // Starts receiver pairing
	MAV_CMD_GET_MESSAGE_INTERVAL              = 510  // Request the interval between messages for a particular MAVLink message ID
	MAV_CMD_SET_MESSAGE_INTERVAL              = 511  // Request the interval between messages for a particular MAVLink message ID
	MAV_CMD_REQUEST_AUTOPILOT_CAPABILITIES    = 520  // Request autopilot capabilities
	MAV_CMD_REQUEST_CAMERA_INFORMATION        = 521  // Request camera information
	MAV_CMD_REQUEST_CAMERA_SETTINGS           = 522  // Request camera settings
	MAV_CMD_REQUEST_STORAGE_INFORMATION       = 525  // Request storage information
	MAV_CMD_STORAGE_FORMAT                    = 526  // Format a storage medium
	MAV_CMD_REQUEST_CAMERA_CAPTURE_STATUS     = 527  // Request camera capture status
	MAV_CMD_REQUEST_FLIGHT_INFORMATION        = 528  // Request flight information
	MAV_CMD_RESET_CAMERA_SETTINGS             = 529  // Reset all camera settings to Factory Default
	MAV_CMD_SET_CAMERA_MODE                   = 530  // Set camera mode
	MAV_CMD_IMAGE_START_CAPTURE               = 2000 // Start image capture sequence
	MAV_CMD_IMAGE_STOP_CAPTURE                = 2001 // Stop image capture sequence
	MAV_CMD_REQUEST_CAMERA_IMAGE_CAPTURE      = 2002 // Re-request a CAMERA_IMAGE_CAPTURED response
	MAV_CMD_DO_TRIGGER_CONTROL                = 2003 // Enable or disable on-board camera triggering system
	MAV_CMD_VIDEO_START_CAPTURE               = 2500 // Start video capture
	MAV_CMD_VIDEO_STOP_CAPTURE                = 2501 // Stop video capture
	MAV_CMD_VIDEO_START_STREAMING             = 2502 // Start video streaming
	MAV_CMD_VIDEO_STOP_STREAMING              = 2503 // Stop video streaming
	MAV_CMD_REQUEST_VIDEO_STREAM_INFORMATION  = 2504 // Request video stream information
	MAV_CMD_REQUEST_VIDEO_STREAM_STATUS       = 2505 // Request video stream status
	MAV_CMD_LOGGING_START                     = 2510 // Request to start streaming logging data
	MAV_CMD_LOGGING_STOP                      = 2511 // Request to stop streaming log data
	MAV_CMD_AIRFRAME_CONFIGURATION            = 2520 // Configure onboard airframe
	MAV_CMD_CONTROL_HIGH_LATENCY              = 2600 // Enable/disable high latency control
	MAV_CMD_PANORAMA_CREATE                   = 2800 // Create a panorama at the current position
	MAV_CMD_DO_VTOL_TRANSITION                = 3000 // Request VTOL transition
	MAV_CMD_ARM_AUTHORIZATION_REQUEST         = 3001 // Request authorization to arm the vehicle
	MAV_CMD_SET_GUIDED_SUBMODE_STANDARD       = 4000 // Set guided submode
	MAV_CMD_SET_GUIDED_SUBMODE_CIRCLE         = 4001 // Set guided submode for circle
	MAV_CMD_CONDITION_GATE                    = 4501 // Delay mission state machine until gate has been reached
	MAV_CMD_NAV_FENCE_RETURN_POINT            = 5000 // Fence return point
	MAV_CMD_NAV_FENCE_POLYGON_VERTEX_INCLUSION = 5001 // Fence vertex for an inclusion polygon
	MAV_CMD_NAV_FENCE_POLYGON_VERTEX_EXCLUSION = 5002 // Fence vertex for an exclusion polygon
	MAV_CMD_NAV_FENCE_CIRCLE_INCLUSION        = 5003 // Circular fence area
	MAV_CMD_NAV_FENCE_CIRCLE_EXCLUSION        = 5004 // Circular fence area
	MAV_CMD_NAV_RALLY_POINT                   = 5100 // Rally point
)

// Battery Status Constants
const (
	// MavBatteryChargeStateUndefined represents battery charge state is not provided
	MavBatteryChargeStateUndefined = 0
	// MavBatteryChargeStateOk represents battery is not in use
	MavBatteryChargeStateOk = 1
	// MavBatteryChargeStateLow represents battery is in low state
	MavBatteryChargeStateLow = 2
	// MavBatteryChargeStateCritical represents battery is in critical state
	MavBatteryChargeStateCritical = 3
	// MavBatteryChargeStateEmergency represents battery is in emergency state
	MavBatteryChargeStateEmergency = 4
	// MavBatteryChargeStateFailed represents battery is damaged
	MavBatteryChargeStateFailed = 5
	// MavBatteryChargeStateUnhealthy represents battery is degraded
	MavBatteryChargeStateUnhealthy = 6
	// MavBatteryChargeStateCharging represents battery is charging
	MavBatteryChargeStateCharging = 7
)

// Geofence Actions
const (
	FENCE_ACTION_NONE          = 0 // Disable fenced mode
	FENCE_ACTION_GUIDED        = 1 // Switched to guided mode to return point (fence point 0)
	FENCE_ACTION_REPORT        = 2 // Report fence breach, but don't take action
	FENCE_ACTION_GUIDED_THR_PASS = 3 // Guided mode but passes through fence
	FENCE_ACTION_RTL           = 4 // Switch to RTL (return to launch) mode and head for the return point
	FENCE_ACTION_HOLD          = 5 // Switch to HOLD mode
	FENCE_ACTION_TERMINATE     = 6 // Terminate flight
	FENCE_ACTION_LAND          = 7 // Switch to Land mode
)

// Robotics-Specific Constants for SemStreams
const (
	// Default message routing prefixes
	ROBOTICS_MESSAGE_PREFIX = "robotics"
	MAVLINK_MESSAGE_PREFIX  = "mavlink"
	
	// Default coordinate system references
	DEFAULT_HOME_LAT = 0.0
	DEFAULT_HOME_LON = 0.0
	DEFAULT_HOME_ALT = 0.0
	
	// Safety thresholds
	DEFAULT_LOW_BATTERY_THRESHOLD      = 20.0  // 20% battery remaining
	DEFAULT_CRITICAL_BATTERY_THRESHOLD = 10.0  // 10% battery remaining
	DEFAULT_EMERGENCY_BATTERY_THRESHOLD = 5.0  // 5% battery remaining
	
	// Geofence default values
	DEFAULT_GEOFENCE_MAX_DISTANCE = 1000.0 // Maximum distance from home in meters
	DEFAULT_GEOFENCE_MAX_ALTITUDE = 400.0  // Maximum altitude in meters (FAA limit)
	
	// Communication timeouts
	DEFAULT_HEARTBEAT_TIMEOUT     = 5.0  // Seconds - consider link lost after this
	DEFAULT_COMMAND_TIMEOUT       = 2.0  // Seconds - timeout for command acknowledgment
	DEFAULT_TELEMETRY_RATE        = 1.0  // Hz - default telemetry publishing rate
	
	// Vehicle capability flags
	MAV_PROTOCOL_CAPABILITY_MISSION_FLOAT            = 1      // Supports float mission items
	MAV_PROTOCOL_CAPABILITY_PARAM_FLOAT              = 2      // Supports float parameter protocol
	MAV_PROTOCOL_CAPABILITY_MISSION_INT              = 4      // Supports mission int protocol
	MAV_PROTOCOL_CAPABILITY_COMMAND_INT              = 8      // Supports command int protocol
	MAV_PROTOCOL_CAPABILITY_PARAM_UNION              = 16     // Supports param union protocol
	MAV_PROTOCOL_CAPABILITY_FTP                      = 32     // Supports file transfer protocol
	MAV_PROTOCOL_CAPABILITY_SET_ATTITUDE_TARGET      = 64     // Supports set attitude target
	MAV_PROTOCOL_CAPABILITY_SET_POSITION_TARGET_LOCAL_NED = 128 // Supports set position target local NED
	MAV_PROTOCOL_CAPABILITY_SET_POSITION_TARGET_GLOBAL_INT = 256 // Supports set position target global int
	MAV_PROTOCOL_CAPABILITY_TERRAIN                  = 512    // Supports terrain protocol / messages
	MAV_PROTOCOL_CAPABILITY_SET_ACTUATOR_TARGET      = 1024   // Supports direct actuator control
	MAV_PROTOCOL_CAPABILITY_FLIGHT_TERMINATION       = 2048   // Supports flight termination
	MAV_PROTOCOL_CAPABILITY_COMPASS_CALIBRATION      = 4096   // Supports compass calibration
	MAV_PROTOCOL_CAPABILITY_MAVLINK2                 = 8192   // Supports MAVLink version 2
	MAV_PROTOCOL_CAPABILITY_MISSION_FENCE            = 16384  // Supports mission fence protocol
	MAV_PROTOCOL_CAPABILITY_MISSION_RALLY            = 32768  // Supports mission rally protocol
	MAV_PROTOCOL_CAPABILITY_FLIGHT_INFORMATION       = 65536  // Supports flight information
)

// GetVehicleTypeName returns the human-readable name for a MAV_TYPE constant
func GetVehicleTypeName(vehicleType uint8) string {
	switch vehicleType {
	case MavTypeGeneric:
		return "Generic"
	case MavTypeFixedWing:
		return "Fixed Wing"
	case MavTypeQuadrotor:
		return "Quadrotor"
	case MavTypeCoaxial:
		return "Coaxial Helicopter"
	case MavTypeHelicopter:
		return "Helicopter"
	case MavTypeAntennaTracker:
		return "Antenna Tracker"
	case MavTypeGcs:
		return "Ground Control Station"
	case MavTypeAirship:
		return "Airship"
	case MavTypeFreeBalloon:
		return "Free Balloon"
	case MavTypeRocket:
		return "Rocket"
	case MavTypeGroundRover:
		return "Ground Rover"
	case MavTypeSurfaceBoat:
		return "Surface Boat"
	case MavTypeSubmarine:
		return "Submarine"
	case MavTypeHexarotor:
		return "Hexarotor"
	case MavTypeOctorotor:
		return "Octorotor"
	case MavTypeTricopter:
		return "Tricopter"
	case MavTypeFlappingWing:
		return "Flapping Wing"
	case MavTypeKite:
		return "Kite"
	case MavTypeOnboardController:
		return "Onboard Controller"
	case MavTypeVtolDuorotor:
		return "VTOL Duorotor"
	case MavTypeVtolQuadrotor:
		return "VTOL Quadrotor"
	case MavTypeVtolTiltrotor:
		return "VTOL Tiltrotor"
	case MavTypeGimbal:
		return "Gimbal"
	case MavTypeAdsb:
		return "ADSB"
	case MavTypeParafoil:
		return "Parafoil"
	case MavTypeDodecarotor:
		return "Dodecarotor"
	case MavTypeCamera:
		return "Camera"
	case MavTypeChargingStation:
		return "Charging Station"
	case MavTypeFlarm:
		return "FLARM"
	case MavTypeServo:
		return "Servo"
	default:
		return "Unknown"
	}
}

// GetAutopilotName returns the human-readable name for a MAV_AUTOPILOT constant
func GetAutopilotName(autopilot uint8) string {
	switch autopilot {
	case MavAutopilotGeneric:
		return "Generic"
	case MavAutopilotReserved:
		return "Reserved"
	case MavAutopilotSlugs:
		return "SLUGS"
	case MavAutopilotArdupilotmega:
		return "ArduPilot"
	case MavAutopilotOpenpilot:
		return "OpenPilot"
	case MavAutopilotGenericWaypointsOnly:
		return "Generic (Waypoints Only)"
	case MavAutopilotGenericWaypointsAndSimpleNavigationOnly:
		return "Generic (Simple Navigation)"
	case MavAutopilotGenericMissionFull:
		return "Generic (Full Mission)"
	case MavAutopilotInvalid:
		return "Invalid/GCS"
	case MavAutopilotPpz:
		return "PPZ"
	case MavAutopilotUdb:
		return "UDB"
	case MavAutopilotFp:
		return "FlexiPilot"
	case MavAutopilotPx4:
		return "PX4"
	case MavAutopilotSmaccmpilot:
		return "SMACCMPilot"
	case MavAutopilotAutoquad:
		return "AutoQuad"
	case MavAutopilotArmazila:
		return "Armazila"
	case MavAutopilotAerob:
		return "Aerob"
	case MavAutopilotAsluav:
		return "ASLUAV"
	case MavAutopilotSmartap:
		return "SmartAP"
	case MavAutopilotAirrails:
		return "AirRails"
	default:
		return "Unknown"
	}
}

// GetSystemStatusName returns the human-readable name for a MAV_STATE constant
func GetSystemStatusName(status uint8) string {
	switch status {
	case MavStateUninit:
		return "Uninitialized"
	case MavStateBoot:
		return "Booting"
	case MavStateCalibrating:
		return "Calibrating"
	case MavStateStandby:
		return "Standby"
	case MavStateActive:
		return "Active"
	case MavStateCritical:
		return "Critical"
	case MavStateEmergency:
		return "Emergency"
	case MavStatePoweroff:
		return "Power Off"
	case MavStateFlightTermination:
		return "Flight Termination"
	default:
		return "Unknown"
	}
}

// GetBatteryChargeStateName returns the human-readable name for battery charge state
func GetBatteryChargeStateName(state uint8) string {
	switch state {
	case MavBatteryChargeStateUndefined:
		return "Undefined"
	case MavBatteryChargeStateOk:
		return "OK"
	case MavBatteryChargeStateLow:
		return "Low"
	case MavBatteryChargeStateCritical:
		return "Critical"
	case MavBatteryChargeStateEmergency:
		return "Emergency"
	case MavBatteryChargeStateFailed:
		return "Failed"
	case MavBatteryChargeStateUnhealthy:
		return "Unhealthy"
	case MavBatteryChargeStateCharging:
		return "Charging"
	default:
		return "Unknown"
	}
}