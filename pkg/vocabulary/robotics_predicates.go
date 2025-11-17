package vocabulary

// Predicate vocabulary using three-level dotted notation: domain.category.property
// This maintains consistency with the unified semantic architecture from Alpha Week 1
// where ALL notation uses dots (no colons anywhere).
//
// Design principles:
//   - Three levels: domain.category.property (e.g., "robotics.battery.level")
//   - Enables NATS wildcard queries: "robotics.battery.*" finds all battery predicates
//   - Human readable: "robotics.flight.armed" is clear and semantic
//   - Domain scoped: Each domain manages its own predicate categories
//   - Consistent with EntityType.Key(), MessageType.Key(), and EntityID.Key() patterns
//
// Predicate naming conventions:
//   - domain: lowercase, represents business domain (robotics, sensors, geo, etc.)
//   - category: lowercase, groups related properties (battery, flight, location, etc.)  
//   - property: lowercase, specific property name (level, armed, latitude, etc.)
//   - No underscores or special characters (dots only for level separation)

// Robotics Domain Predicates
// These predicates describe properties and relationships of robotic systems

const (
	// Battery category - power and energy related properties
	ROBOTICS_BATTERY_LEVEL      = "robotics.battery.level"      // float64, 0-100 percentage
	ROBOTICS_BATTERY_VOLTAGE    = "robotics.battery.voltage"    // float64, volts
	ROBOTICS_BATTERY_CURRENT    = "robotics.battery.current"    // float64, amps  
	ROBOTICS_BATTERY_REMAINING  = "robotics.battery.remaining"  // float64, minutes estimated
	ROBOTICS_BATTERY_CONSUMED   = "robotics.battery.consumed"   // float64, mAh consumed

	// Flight category - flight state and control properties
	ROBOTICS_FLIGHT_MODE        = "robotics.flight.mode"        // string, flight mode name
	ROBOTICS_FLIGHT_ARMED       = "robotics.flight.armed"       // bool, armed state
	ROBOTICS_FLIGHT_GUIDED      = "robotics.flight.guided"      // bool, guided mode active
	ROBOTICS_FLIGHT_STABILIZE   = "robotics.flight.stabilize"   // bool, stabilization active
	ROBOTICS_FLIGHT_AUTO        = "robotics.flight.auto"        // bool, autonomous mode

	// System category - system identification and status  
	ROBOTICS_SYSTEM_ID          = "robotics.system.id"          // string, system identifier
	ROBOTICS_SYSTEM_STATUS      = "robotics.system.status"      // string, system status
	ROBOTICS_SYSTEM_TYPE        = "robotics.system.type"        // string, system type
	ROBOTICS_SYSTEM_VERSION     = "robotics.system.version"     // string, firmware/software version
	ROBOTICS_SYSTEM_UPTIME      = "robotics.system.uptime"      // float64, seconds since boot

	// Component category - relationships and part identification
	ROBOTICS_COMPONENT_HAS      = "robotics.component.has"      // entity reference, has component
	ROBOTICS_COMPONENT_POWERED  = "robotics.component.powered"  // entity reference, powers component  
	ROBOTICS_COMPONENT_MONITORS = "robotics.component.monitors" // entity reference, monitors component
	ROBOTICS_COMPONENT_CONTROLS = "robotics.component.controls" // entity reference, controls component

	// Mission category - mission and task related properties
	ROBOTICS_MISSION_CURRENT    = "robotics.mission.current"    // string, current mission ID
	ROBOTICS_MISSION_WAYPOINT   = "robotics.mission.waypoint"   // int, current waypoint number
	ROBOTICS_MISSION_PROGRESS   = "robotics.mission.progress"   // float64, 0-100 percentage complete
	ROBOTICS_MISSION_STATUS     = "robotics.mission.status"     // string, mission status

	// Attitude category - orientation and rotation
	ROBOTICS_ATTITUDE_ROLL      = "robotics.attitude.roll"      // float64, radians
	ROBOTICS_ATTITUDE_PITCH     = "robotics.attitude.pitch"     // float64, radians  
	ROBOTICS_ATTITUDE_YAW       = "robotics.attitude.yaw"       // float64, radians
	ROBOTICS_ATTITUDE_ROLLSPEED = "robotics.attitude.rollspeed" // float64, rad/s
	ROBOTICS_ATTITUDE_PITCHSPEED = "robotics.attitude.pitchspeed" // float64, rad/s
	ROBOTICS_ATTITUDE_YAWSPEED  = "robotics.attitude.yawspeed"  // float64, rad/s
)

// Sensor Domain Predicates  
// These predicates describe sensor measurements and environmental data

const (
	// Temperature category
	SENSOR_TEMPERATURE_CELSIUS    = "sensor.temperature.celsius"    // float64, degrees Celsius
	SENSOR_TEMPERATURE_FAHRENHEIT = "sensor.temperature.fahrenheit" // float64, degrees Fahrenheit
	SENSOR_TEMPERATURE_KELVIN     = "sensor.temperature.kelvin"     // float64, degrees Kelvin

	// Pressure category  
	SENSOR_PRESSURE_PASCALS       = "sensor.pressure.pascals"       // float64, pascals
	SENSOR_PRESSURE_BAR          = "sensor.pressure.bar"           // float64, bar
	SENSOR_PRESSURE_PSI          = "sensor.pressure.psi"           // float64, pounds per square inch

	// Humidity category
	SENSOR_HUMIDITY_PERCENT      = "sensor.humidity.percent"       // float64, 0-100 percentage
	SENSOR_HUMIDITY_ABSOLUTE     = "sensor.humidity.absolute"      // float64, g/m³

	// Acceleration category (IMU sensors)
	SENSOR_ACCEL_X               = "sensor.accel.x"                // float64, m/s²
	SENSOR_ACCEL_Y               = "sensor.accel.y"                // float64, m/s²
	SENSOR_ACCEL_Z               = "sensor.accel.z"                // float64, m/s²

	// Gyroscope category (IMU sensors)
	SENSOR_GYRO_X                = "sensor.gyro.x"                 // float64, rad/s
	SENSOR_GYRO_Y                = "sensor.gyro.y"                 // float64, rad/s
	SENSOR_GYRO_Z                = "sensor.gyro.z"                 // float64, rad/s

	// Magnetometer category (IMU sensors)
	SENSOR_MAG_X                 = "sensor.mag.x"                  // float64, gauss
	SENSOR_MAG_Y                 = "sensor.mag.y"                  // float64, gauss
	SENSOR_MAG_Z                 = "sensor.mag.z"                  // float64, gauss
)

// Geospatial Domain Predicates
// These predicates describe location, positioning, and geographic data

const (
	// Location category (WGS84 coordinates)
	GEO_LOCATION_LATITUDE        = "geo.location.latitude"         // float64, degrees (-90 to 90)
	GEO_LOCATION_LONGITUDE       = "geo.location.longitude"        // float64, degrees (-180 to 180)
	GEO_LOCATION_ALTITUDE        = "geo.location.altitude"         // float64, meters above sea level
	GEO_LOCATION_ELEVATION       = "geo.location.elevation"        // float64, meters above ground

	// Velocity category (movement in geographic coordinates)
	GEO_VELOCITY_GROUND          = "geo.velocity.ground"           // float64, m/s ground speed
	GEO_VELOCITY_VERTICAL        = "geo.velocity.vertical"         // float64, m/s climb/descent rate  
	GEO_VELOCITY_HEADING         = "geo.velocity.heading"          // float64, degrees (0-360)

	// Accuracy category (GPS/positioning accuracy)
	GEO_ACCURACY_HORIZONTAL      = "geo.accuracy.horizontal"       // float64, meters CEP
	GEO_ACCURACY_VERTICAL        = "geo.accuracy.vertical"         // float64, meters vertical accuracy
	GEO_ACCURACY_DILUTION        = "geo.accuracy.dilution"         // float64, dilution of precision

	// Zone category (geographic regions and areas)
	GEO_ZONE_UTM                 = "geo.zone.utm"                  // string, UTM zone
	GEO_ZONE_MGRS                = "geo.zone.mgrs"                 // string, MGRS grid
	GEO_ZONE_REGION              = "geo.zone.region"               // string, geographic region name
)

// Temporal Domain Predicates
// These predicates describe time-related properties and lifecycle events

const (
	// Lifecycle category - entity lifecycle events
	TIME_LIFECYCLE_CREATED       = "time.lifecycle.created"        // time.Time, when entity was created
	TIME_LIFECYCLE_UPDATED       = "time.lifecycle.updated"        // time.Time, when entity was last updated  
	TIME_LIFECYCLE_SEEN          = "time.lifecycle.seen"           // time.Time, when entity was last observed
	TIME_LIFECYCLE_EXPIRED       = "time.lifecycle.expired"        // time.Time, when entity expired/deleted

	// Duration category - time periods and intervals
	TIME_DURATION_ACTIVE         = "time.duration.active"          // float64, seconds active
	TIME_DURATION_IDLE           = "time.duration.idle"            // float64, seconds idle
	TIME_DURATION_TOTAL          = "time.duration.total"           // float64, total seconds

	// Schedule category - planned and scheduled events
	TIME_SCHEDULE_START          = "time.schedule.start"           // time.Time, scheduled start
	TIME_SCHEDULE_END            = "time.schedule.end"             // time.Time, scheduled end
	TIME_SCHEDULE_NEXT           = "time.schedule.next"            // time.Time, next scheduled event
)

// Network Domain Predicates  
// These predicates describe network connectivity and communication

const (
	// Connection category - network connection properties
	NETWORK_CONNECTION_STATUS    = "network.connection.status"     // string, connection status
	NETWORK_CONNECTION_STRENGTH  = "network.connection.strength"   // float64, signal strength
	NETWORK_CONNECTION_LATENCY   = "network.connection.latency"    // float64, milliseconds

	// Protocol category - communication protocol information
	NETWORK_PROTOCOL_TYPE        = "network.protocol.type"         // string, protocol name
	NETWORK_PROTOCOL_VERSION     = "network.protocol.version"      // string, protocol version
	NETWORK_PROTOCOL_PORT        = "network.protocol.port"         // int, port number

	// Traffic category - data transfer metrics
	NETWORK_TRAFFIC_BYTES_IN     = "network.traffic.bytes.in"      // int64, bytes received
	NETWORK_TRAFFIC_BYTES_OUT    = "network.traffic.bytes.out"     // int64, bytes sent
	NETWORK_TRAFFIC_PACKETS_IN   = "network.traffic.packets.in"    // int64, packets received
	NETWORK_TRAFFIC_PACKETS_OUT  = "network.traffic.packets.out"   // int64, packets sent
)

// Quality Domain Predicates
// These predicates describe data quality, confidence, and validation

const (
	// Confidence category - certainty and reliability metrics
	QUALITY_CONFIDENCE_SCORE     = "quality.confidence.score"      // float64, 0-1 confidence level
	QUALITY_CONFIDENCE_SOURCE    = "quality.confidence.source"     // string, source of confidence assessment
	QUALITY_CONFIDENCE_METHOD    = "quality.confidence.method"     // string, confidence calculation method

	// Validation category - data validation and verification
	QUALITY_VALIDATION_STATUS    = "quality.validation.status"     // string, validation status
	QUALITY_VALIDATION_ERRORS    = "quality.validation.errors"     // int, number of validation errors
	QUALITY_VALIDATION_WARNINGS  = "quality.validation.warnings"   // int, number of warnings

	// Accuracy category - measurement accuracy and precision
	QUALITY_ACCURACY_ABSOLUTE    = "quality.accuracy.absolute"     // float64, absolute accuracy
	QUALITY_ACCURACY_RELATIVE    = "quality.accuracy.relative"     // float64, relative accuracy percentage
	QUALITY_ACCURACY_PRECISION   = "quality.accuracy.precision"    // float64, measurement precision
)

// PredicateMetadata provides semantic information about each predicate
// This enables validation, type checking, and documentation generation
type PredicateMetadata struct {
	// Name is the predicate constant (e.g., "robotics.battery.level")
	Name string
	// Description provides human-readable documentation
	Description string
	// DataType indicates the expected Go type for the object value
	DataType string
	// Units specifies the measurement units (if applicable)
	Units string
	// Range describes valid value ranges (if applicable)  
	Range string
	// Domain identifies which domain owns this predicate
	Domain string
	// Category identifies the predicate category within the domain
	Category string
}

// GetPredicateMetadata returns metadata for well-known predicates
// This enables validation and documentation generation
func GetPredicateMetadata(predicate string) *PredicateMetadata {
	metadata := map[string]PredicateMetadata{
		ROBOTICS_BATTERY_LEVEL: {
			Name:        ROBOTICS_BATTERY_LEVEL,
			Description: "Battery charge level as a percentage",
			DataType:    "float64",
			Units:       "percentage",
			Range:       "0-100",
			Domain:      "robotics",
			Category:    "battery",
		},
		ROBOTICS_FLIGHT_ARMED: {
			Name:        ROBOTICS_FLIGHT_ARMED,
			Description: "Whether the vehicle is armed for flight",
			DataType:    "bool",
			Units:       "",
			Range:       "true/false",
			Domain:      "robotics",
			Category:    "flight",
		},
		GEO_LOCATION_LATITUDE: {
			Name:        GEO_LOCATION_LATITUDE,
			Description: "WGS84 latitude coordinate",
			DataType:    "float64",
			Units:       "degrees",
			Range:       "-90 to 90",
			Domain:      "geo",
			Category:    "location",
		},
		GEO_LOCATION_LONGITUDE: {
			Name:        GEO_LOCATION_LONGITUDE,
			Description: "WGS84 longitude coordinate",
			DataType:    "float64", 
			Units:       "degrees",
			Range:       "-180 to 180",
			Domain:      "geo",
			Category:    "location",
		},
		// Add more metadata as needed
	}
	
	if meta, exists := metadata[predicate]; exists {
		return &meta
	}
	return nil
}

// IsValidPredicate checks if a predicate follows the three-level dotted notation
// and matches the expected format: domain.category.property
func IsValidPredicate(predicate string) bool {
	if predicate == "" {
		return false
	}
	
	// Count dots to ensure three-level structure
	dotCount := 0
	for _, char := range predicate {
		if char == '.' {
			dotCount++
		}
	}
	
	// Must have exactly 2 dots for three levels
	return dotCount == 2
}