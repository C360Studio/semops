//go:build ignore
// +build ignore

package robotics

// Domain constant
const Domain = "robotics"

// Semantic robotics subjects
const (
	ProcessedHeartbeat = "semantic.robotics.heartbeat"
	ProcessedPosition  = "semantic.robotics.position"
	ProcessedAttitude  = "semantic.robotics.attitude"
	ProcessedBattery   = "semantic.robotics.battery"
	ProcessedMission   = "semantic.robotics.mission"
)

// Event robotics subjects
const (
	EventDeployment = "events.robotics.deployment"
	EventEmergency  = "events.robotics.emergency"
	EventFormation  = "events.robotics.formation"
)

// Entity subjects for robotics
const (
	EntityVehicle = "entities.robotics.vehicle"
	EntityFleet   = "entities.robotics.fleet"
)

// RegisterSubjects returns all subjects owned by the robotics plugin
func RegisterSubjects() []string {
	return []string{
		ProcessedHeartbeat,
		ProcessedPosition,
		ProcessedAttitude,
		ProcessedBattery,
		ProcessedMission,
		EventDeployment,
		EventEmergency,
		EventFormation,
		EntityVehicle,
		EntityFleet,
	}
}

// AllProcessedSubjects returns only processed message subjects
func AllProcessedSubjects() []string {
	return []string{
		ProcessedHeartbeat,
		ProcessedPosition,
		ProcessedAttitude,
		ProcessedBattery,
		ProcessedMission,
	}
}

// AllEventSubjects returns only event subjects
func AllEventSubjects() []string {
	return []string{
		EventDeployment,
		EventEmergency,
		EventFormation,
	}
}

// AllEntitySubjects returns only entity subjects
func AllEntitySubjects() []string {
	return []string{
		EntityVehicle,
		EntityFleet,
	}
}
