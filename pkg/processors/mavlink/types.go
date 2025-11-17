package robotics

import message "github.com/c360/semstreams/message"

// Message types for the robotics domain
var (
	// Position represents robotic platform position data
	Position = message.Type{
		Domain:   "robotics",
		Category: "position",
		Version:  "v1",
	}

	// Heartbeat represents robotic platform heartbeat/status data
	Heartbeat = message.Type{
		Domain:   "robotics",
		Category: "heartbeat",
		Version:  "v1",
	}

	// Mission represents robotic mission commands and status
	Mission = message.Type{
		Domain:   "robotics",
		Category: "mission",
		Version:  "v1",
	}

	// Command represents robotic control commands
	Command = message.Type{
		Domain:   "robotics",
		Category: "command",
		Version:  "v1",
	}

	// Attitude represents robotic platform orientation data
	Attitude = message.Type{
		Domain:   "robotics",
		Category: "attitude",
		Version:  "v1",
	}

	// Battery represents robotic platform battery status
	Battery = message.Type{
		Domain:   "robotics",
		Category: "battery",
		Version:  "v1",
	}
)

// Domain-specific entity types using structured EntityType (replaces old colon notation)
var (
	// Vehicle types
	DomainTypeUAV        = message.EntityType{Domain: "robotics", Type: "uav"}
	DomainTypeUGV        = message.EntityType{Domain: "robotics", Type: "ugv"}
	DomainTypeUSV        = message.EntityType{Domain: "robotics", Type: "usv"}
	DomainTypeAUV        = message.EntityType{Domain: "robotics", Type: "auv"}
	DomainTypeRover      = message.EntityType{Domain: "robotics", Type: "rover"}
	DomainTypeDrone      = message.EntityType{Domain: "robotics", Type: "drone"}

	// System components
	DomainTypeAutopilot  = message.EntityType{Domain: "robotics", Type: "autopilot"}
	DomainTypeGimbal     = message.EntityType{Domain: "robotics", Type: "gimbal"}
	DomainTypeActuator   = message.EntityType{Domain: "robotics", Type: "actuator"}
	DomainTypeSensor     = message.EntityType{Domain: "robotics", Type: "sensor"}

	// Mission/event types
	DomainTypeMission    = message.EntityType{Domain: "robotics", Type: "mission"}
	DomainTypeWaypoint   = message.EntityType{Domain: "robotics", Type: "waypoint"}
	DomainTypeCommand    = message.EntityType{Domain: "robotics", Type: "command"}
	DomainTypeTelemetry  = message.EntityType{Domain: "robotics", Type: "telemetry"}
)