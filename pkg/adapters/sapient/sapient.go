// Package sapient implements the first SAPIENT feed boundary for SemOps.
//
// The current boundary provides JSON and descriptor-based protobuf preflight
// for representative BSI Flex 335 v2 message shapes, plus raw replay fixtures
// for repeatable component tests. It does not claim Dstl harness compliance,
// product SAPIENT support, tasking authority, or full-message coverage.
package sapient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const StandardVersion = "BSI Flex 335 v2.0"

type ContentKind string

const (
	ContentDetectionReport ContentKind = "detectionReport"
	ContentRegistration    ContentKind = "registration"
	ContentStatusReport    ContentKind = "statusReport"
	ContentTaskAck         ContentKind = "taskAck"
)

type Message struct {
	Timestamp       time.Time
	NodeID          string
	DestinationID   *string
	Content         ContentKind
	DetectionReport *DetectionReport
	Registration    *Registration
	StatusReport    *StatusReport
	TaskAck         *TaskAck
}

type DetectionReport struct {
	ReportID            string
	ObjectID            string
	TaskID              *string
	State               string
	Location            *Location
	RangeBearing        *RangeBearing
	DetectionConfidence *float64
	Classifications     []Classification
	AssociatedDetection []DetectionRef
	DerivedDetection    []DetectionRef
	HasENUVelocity      bool
	Colour              string
	ID                  string
}

type Location struct {
	X                float64
	Y                float64
	Z                *float64
	CoordinateSystem string
	Datum            string
	UTMZone          string
}

type RangeBearing struct {
	Elevation        *float64
	Azimuth          *float64
	Range            *float64
	CoordinateSystem string
	Datum            string
}

type Classification struct {
	Type       string
	Confidence *float64
	SubClass   []SubClass
}

type SubClass struct {
	Type       string
	Confidence *float64
	Level      int
	SubClass   []SubClass
}

type DetectionRef struct {
	Timestamp       *time.Time
	NodeID          string
	ObjectID        string
	AssociationType string
}

type Registration struct {
	ICDVersion   string
	NodeTypes    []string
	Capabilities []Capability
	Name         string
	ShortName    string
}

type Capability struct {
	Category string
	Type     string
	Value    string
	Units    string
}

type StatusReport struct {
	ReportID     string
	System       string
	Info         string
	ActiveTaskID *string
	Mode         string
	NodeLocation *Location
}

type TaskAck struct {
	TaskID string
	Status string
	Reason []string
}

type object map[string]json.RawMessage

var (
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	ulidPattern = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
)

func ParseJSONMessage(data []byte) (Message, error) {
	raw, err := decodeObject(data)
	if err != nil {
		return Message{}, err
	}
	timestamp, err := requiredTime(raw, "timestamp")
	if err != nil {
		return Message{}, err
	}
	nodeID, err := requiredUUID(raw, "nodeId")
	if err != nil {
		return Message{}, err
	}
	destinationID, err := optionalUUID(raw, "destinationId")
	if err != nil {
		return Message{}, err
	}
	content, payload, err := findContent(raw)
	if err != nil {
		return Message{}, err
	}

	msg := Message{
		Timestamp:     timestamp,
		NodeID:        nodeID,
		DestinationID: destinationID,
		Content:       content,
	}
	switch content {
	case ContentDetectionReport:
		parsed, err := parseDetectionReport(payload)
		if err != nil {
			return Message{}, fmt.Errorf("sapient detectionReport: %w", err)
		}
		msg.DetectionReport = &parsed
	case ContentRegistration:
		parsed, err := parseRegistration(payload)
		if err != nil {
			return Message{}, fmt.Errorf("sapient registration: %w", err)
		}
		msg.Registration = &parsed
	case ContentStatusReport:
		parsed, err := parseStatusReport(payload)
		if err != nil {
			return Message{}, fmt.Errorf("sapient statusReport: %w", err)
		}
		msg.StatusReport = &parsed
	case ContentTaskAck:
		parsed, err := parseTaskAck(payload)
		if err != nil {
			return Message{}, fmt.Errorf("sapient taskAck: %w", err)
		}
		msg.TaskAck = &parsed
	default:
		return Message{}, fmt.Errorf("sapient content %q has no preflight parser", content)
	}
	return msg, nil
}

func parseDetectionReport(raw json.RawMessage) (DetectionReport, error) {
	obj, err := decodeRawObject(raw, "detectionReport")
	if err != nil {
		return DetectionReport{}, err
	}
	reportID, err := requiredULID(obj, "reportId")
	if err != nil {
		return DetectionReport{}, err
	}
	objectID, err := requiredULID(obj, "objectId")
	if err != nil {
		return DetectionReport{}, err
	}
	taskID, err := optionalULID(obj, "taskId")
	if err != nil {
		return DetectionReport{}, err
	}
	location, rangeBearing, err := parseDetectionLocation(obj)
	if err != nil {
		return DetectionReport{}, err
	}
	confidence, err := optionalNumber(obj, "detectionConfidence")
	if err != nil {
		return DetectionReport{}, err
	}
	if confidence != nil && (*confidence < 0 || *confidence > 1) {
		return DetectionReport{}, errors.New("detectionConfidence must be between 0 and 1")
	}
	classifications, err := parseClassifications(obj["classification"])
	if err != nil {
		return DetectionReport{}, err
	}
	associated, err := parseDetectionRefs(obj["associatedDetection"], true)
	if err != nil {
		return DetectionReport{}, fmt.Errorf("associatedDetection: %w", err)
	}
	derived, err := parseDetectionRefs(obj["derivedDetection"], false)
	if err != nil {
		return DetectionReport{}, fmt.Errorf("derivedDetection: %w", err)
	}
	state, err := optionalString(obj, "state")
	if err != nil {
		return DetectionReport{}, err
	}
	colour, err := optionalString(obj, "colour")
	if err != nil {
		return DetectionReport{}, err
	}
	id, err := optionalString(obj, "id")
	if err != nil {
		return DetectionReport{}, err
	}
	_, hasENUVelocity := obj["enu_velocity"]
	if !hasENUVelocity {
		_, hasENUVelocity = obj["enuVelocity"]
	}

	return DetectionReport{
		ReportID:            reportID,
		ObjectID:            objectID,
		TaskID:              taskID,
		State:               valueOrEmpty(state),
		Location:            location,
		RangeBearing:        rangeBearing,
		DetectionConfidence: confidence,
		Classifications:     classifications,
		AssociatedDetection: associated,
		DerivedDetection:    derived,
		HasENUVelocity:      hasENUVelocity,
		Colour:              valueOrEmpty(colour),
		ID:                  valueOrEmpty(id),
	}, nil
}

func parseRegistration(raw json.RawMessage) (Registration, error) {
	obj, err := decodeRawObject(raw, "registration")
	if err != nil {
		return Registration{}, err
	}
	icdVersion, err := requiredString(obj, "icdVersion")
	if err != nil {
		return Registration{}, err
	}
	if icdVersion != StandardVersion {
		return Registration{}, fmt.Errorf("icdVersion = %q, want %q", icdVersion, StandardVersion)
	}
	nodeDefs, err := requiredArray(obj, "nodeDefinition")
	if err != nil {
		return Registration{}, err
	}
	nodeTypes := make([]string, 0, len(nodeDefs))
	for i, rawNode := range nodeDefs {
		node, err := decodeRawObject(rawNode, fmt.Sprintf("nodeDefinition[%d]", i))
		if err != nil {
			return Registration{}, err
		}
		nodeType, err := requiredEnum(node, "nodeType")
		if err != nil {
			return Registration{}, fmt.Errorf("nodeDefinition[%d]: %w", i, err)
		}
		nodeTypes = append(nodeTypes, nodeType)
	}
	capabilities, err := parseCapabilities(obj)
	if err != nil {
		return Registration{}, err
	}
	if err := validateStatusDefinition(obj["statusDefinition"]); err != nil {
		return Registration{}, fmt.Errorf("statusDefinition: %w", err)
	}
	if err := validateModeDefinitions(obj["modeDefinition"]); err != nil {
		return Registration{}, err
	}
	if err := validateConfigData(obj["configData"]); err != nil {
		return Registration{}, err
	}
	name, err := optionalString(obj, "name")
	if err != nil {
		return Registration{}, err
	}
	shortName, err := optionalString(obj, "shortName")
	if err != nil {
		return Registration{}, err
	}
	return Registration{
		ICDVersion:   icdVersion,
		NodeTypes:    nodeTypes,
		Capabilities: capabilities,
		Name:         valueOrEmpty(name),
		ShortName:    valueOrEmpty(shortName),
	}, nil
}

func parseStatusReport(raw json.RawMessage) (StatusReport, error) {
	obj, err := decodeRawObject(raw, "statusReport")
	if err != nil {
		return StatusReport{}, err
	}
	reportID, err := requiredULID(obj, "reportId")
	if err != nil {
		return StatusReport{}, err
	}
	system, err := requiredEnum(obj, "system")
	if err != nil {
		return StatusReport{}, err
	}
	info, err := requiredEnum(obj, "info")
	if err != nil {
		return StatusReport{}, err
	}
	mode, err := requiredString(obj, "mode")
	if err != nil {
		return StatusReport{}, err
	}
	activeTaskID, err := optionalULID(obj, "activeTaskId")
	if err != nil {
		return StatusReport{}, err
	}
	var nodeLocation *Location
	if rawLocation, ok := obj["nodeLocation"]; ok {
		location, err := parseLocation(rawLocation)
		if err != nil {
			return StatusReport{}, fmt.Errorf("nodeLocation: %w", err)
		}
		nodeLocation = &location
	}
	return StatusReport{
		ReportID:     reportID,
		System:       system,
		Info:         info,
		ActiveTaskID: activeTaskID,
		Mode:         mode,
		NodeLocation: nodeLocation,
	}, nil
}

func parseTaskAck(raw json.RawMessage) (TaskAck, error) {
	obj, err := decodeRawObject(raw, "taskAck")
	if err != nil {
		return TaskAck{}, err
	}
	taskID, err := requiredULID(obj, "taskId")
	if err != nil {
		return TaskAck{}, err
	}
	status, err := requiredEnum(obj, "taskStatus")
	if err != nil {
		return TaskAck{}, err
	}
	reason, err := optionalStringArray(obj, "reason")
	if err != nil {
		return TaskAck{}, err
	}
	return TaskAck{TaskID: taskID, Status: status, Reason: reason}, nil
}

func findContent(raw object) (ContentKind, json.RawMessage, error) {
	known := []ContentKind{
		ContentRegistration,
		"registrationAck",
		ContentStatusReport,
		ContentDetectionReport,
		"task",
		ContentTaskAck,
		"alert",
		"alertAck",
		"error",
	}
	found := make([]ContentKind, 0, 1)
	payloads := make(map[ContentKind]json.RawMessage)
	for _, kind := range known {
		if payload, ok := raw[string(kind)]; ok {
			found = append(found, kind)
			payloads[kind] = payload
		}
	}
	if len(found) == 0 {
		return "", nil, errors.New("sapient message requires exactly one content field")
	}
	if len(found) > 1 {
		return "", nil, fmt.Errorf("sapient message has multiple content fields: %s", joinContent(found))
	}
	return found[0], payloads[found[0]], nil
}

func parseDetectionLocation(obj object) (*Location, *RangeBearing, error) {
	rawLocation, hasLocation := obj["location"]
	rawRangeBearing, hasRangeBearing := obj["rangeBearing"]
	if hasLocation == hasRangeBearing {
		return nil, nil, errors.New("exactly one of location or rangeBearing is required")
	}
	if hasLocation {
		location, err := parseLocation(rawLocation)
		if err != nil {
			return nil, nil, err
		}
		return &location, nil, nil
	}
	rangeBearing, err := parseRangeBearing(rawRangeBearing)
	if err != nil {
		return nil, nil, err
	}
	return nil, &rangeBearing, nil
}

func parseLocation(raw json.RawMessage) (Location, error) {
	obj, err := decodeRawObject(raw, "location")
	if err != nil {
		return Location{}, err
	}
	x, err := requiredNumber(obj, "x")
	if err != nil {
		return Location{}, err
	}
	y, err := requiredNumber(obj, "y")
	if err != nil {
		return Location{}, err
	}
	z, err := optionalNumber(obj, "z")
	if err != nil {
		return Location{}, err
	}
	coordinateSystem, err := requiredEnum(obj, "coordinateSystem")
	if err != nil {
		return Location{}, err
	}
	datum, err := requiredEnum(obj, "datum")
	if err != nil {
		return Location{}, err
	}
	utmZone, err := optionalString(obj, "utmZone")
	if err != nil {
		return Location{}, err
	}
	return Location{
		X:                x,
		Y:                y,
		Z:                z,
		CoordinateSystem: coordinateSystem,
		Datum:            datum,
		UTMZone:          valueOrEmpty(utmZone),
	}, nil
}

func parseRangeBearing(raw json.RawMessage) (RangeBearing, error) {
	obj, err := decodeRawObject(raw, "rangeBearing")
	if err != nil {
		return RangeBearing{}, err
	}
	elevation, err := optionalNumber(obj, "elevation")
	if err != nil {
		return RangeBearing{}, err
	}
	azimuth, err := optionalNumber(obj, "azimuth")
	if err != nil {
		return RangeBearing{}, err
	}
	distance, err := optionalNumber(obj, "range")
	if err != nil {
		return RangeBearing{}, err
	}
	coordinateSystem, err := requiredEnum(obj, "coordinateSystem")
	if err != nil {
		return RangeBearing{}, err
	}
	datum, err := requiredEnum(obj, "datum")
	if err != nil {
		return RangeBearing{}, err
	}
	return RangeBearing{
		Elevation:        elevation,
		Azimuth:          azimuth,
		Range:            distance,
		CoordinateSystem: coordinateSystem,
		Datum:            datum,
	}, nil
}

func parseClassifications(raw json.RawMessage) ([]Classification, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	items, err := decodeRawArray(raw, "classification")
	if err != nil {
		return nil, err
	}
	out := make([]Classification, 0, len(items))
	for i, item := range items {
		obj, err := decodeRawObject(item, fmt.Sprintf("classification[%d]", i))
		if err != nil {
			return nil, err
		}
		classType, err := requiredString(obj, "type")
		if err != nil {
			return nil, fmt.Errorf("classification[%d]: %w", i, err)
		}
		confidence, err := optionalNumber(obj, "confidence")
		if err != nil {
			return nil, fmt.Errorf("classification[%d]: %w", i, err)
		}
		subClass, err := parseSubClasses(obj["subClass"], fmt.Sprintf("classification[%d].subClass", i))
		if err != nil {
			return nil, err
		}
		out = append(out, Classification{Type: classType, Confidence: confidence, SubClass: subClass})
	}
	return out, nil
}

func parseSubClasses(raw json.RawMessage, field string) ([]SubClass, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	items, err := decodeRawArray(raw, field)
	if err != nil {
		return nil, err
	}
	out := make([]SubClass, 0, len(items))
	for i, item := range items {
		obj, err := decodeRawObject(item, fmt.Sprintf("%s[%d]", field, i))
		if err != nil {
			return nil, err
		}
		subType, err := requiredString(obj, "type")
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", field, i, err)
		}
		levelFloat, err := requiredNumber(obj, "level")
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", field, i, err)
		}
		level := int(levelFloat)
		if float64(level) != levelFloat {
			return nil, fmt.Errorf("%s[%d]: level must be an integer", field, i)
		}
		confidence, err := optionalNumber(obj, "confidence")
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", field, i, err)
		}
		children, err := parseSubClasses(obj["subClass"], fmt.Sprintf("%s[%d].subClass", field, i))
		if err != nil {
			return nil, err
		}
		out = append(out, SubClass{Type: subType, Confidence: confidence, Level: level, SubClass: children})
	}
	return out, nil
}

func parseDetectionRefs(raw json.RawMessage, hasAssociationType bool) ([]DetectionRef, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	items, err := decodeRawArray(raw, "detection refs")
	if err != nil {
		return nil, err
	}
	out := make([]DetectionRef, 0, len(items))
	for i, item := range items {
		obj, err := decodeRawObject(item, fmt.Sprintf("ref[%d]", i))
		if err != nil {
			return nil, err
		}
		timestamp, err := optionalTime(obj, "timestamp")
		if err != nil {
			return nil, fmt.Errorf("ref[%d]: %w", i, err)
		}
		nodeID, err := requiredUUID(obj, "nodeId")
		if err != nil {
			return nil, fmt.Errorf("ref[%d]: %w", i, err)
		}
		objectID, err := requiredULID(obj, "objectId")
		if err != nil {
			return nil, fmt.Errorf("ref[%d]: %w", i, err)
		}
		associationType := ""
		if hasAssociationType {
			associationType, err = optionalEnum(obj, "associationType")
			if err != nil {
				return nil, fmt.Errorf("ref[%d]: %w", i, err)
			}
		}
		out = append(out, DetectionRef{
			Timestamp:       timestamp,
			NodeID:          nodeID,
			ObjectID:        objectID,
			AssociationType: associationType,
		})
	}
	return out, nil
}

func parseCapabilities(obj object) ([]Capability, error) {
	items, err := requiredArray(obj, "capabilities")
	if err != nil {
		return nil, err
	}
	out := make([]Capability, 0, len(items))
	for i, item := range items {
		capability, err := decodeRawObject(item, fmt.Sprintf("capabilities[%d]", i))
		if err != nil {
			return nil, err
		}
		category, err := requiredString(capability, "category")
		if err != nil {
			return nil, fmt.Errorf("capabilities[%d]: %w", i, err)
		}
		capType, err := requiredString(capability, "type")
		if err != nil {
			return nil, fmt.Errorf("capabilities[%d]: %w", i, err)
		}
		value, err := optionalString(capability, "value")
		if err != nil {
			return nil, fmt.Errorf("capabilities[%d]: %w", i, err)
		}
		units, err := optionalString(capability, "units")
		if err != nil {
			return nil, fmt.Errorf("capabilities[%d]: %w", i, err)
		}
		out = append(out, Capability{
			Category: category,
			Type:     capType,
			Value:    valueOrEmpty(value),
			Units:    valueOrEmpty(units),
		})
	}
	return out, nil
}

func validateStatusDefinition(raw json.RawMessage) error {
	obj, err := decodeRawObject(raw, "statusDefinition")
	if err != nil {
		return err
	}
	duration, err := decodeRawObject(obj["statusInterval"], "statusInterval")
	if err != nil {
		return err
	}
	return validateDuration(duration)
}

func validateModeDefinitions(raw json.RawMessage) error {
	items, err := decodeRawArray(raw, "modeDefinition")
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return errors.New("modeDefinition requires at least one item")
	}
	for i, item := range items {
		mode, err := decodeRawObject(item, fmt.Sprintf("modeDefinition[%d]", i))
		if err != nil {
			return err
		}
		if _, err := requiredString(mode, "modeName"); err != nil {
			return fmt.Errorf("modeDefinition[%d]: %w", i, err)
		}
		if _, err := requiredEnum(mode, "modeType"); err != nil {
			return fmt.Errorf("modeDefinition[%d]: %w", i, err)
		}
		settleTime, err := decodeRawObject(mode["settleTime"], "settleTime")
		if err != nil {
			return fmt.Errorf("modeDefinition[%d]: %w", i, err)
		}
		if err := validateDuration(settleTime); err != nil {
			return fmt.Errorf("modeDefinition[%d].settleTime: %w", i, err)
		}
		task, err := decodeRawObject(mode["task"], "task")
		if err != nil {
			return fmt.Errorf("modeDefinition[%d]: %w", i, err)
		}
		if _, ok := task["regionDefinition"]; !ok {
			return fmt.Errorf("modeDefinition[%d].task: regionDefinition is required", i)
		}
	}
	return nil
}

func validateConfigData(raw json.RawMessage) error {
	items, err := decodeRawArray(raw, "configData")
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return errors.New("configData requires at least one item")
	}
	for i, item := range items {
		config, err := decodeRawObject(item, fmt.Sprintf("configData[%d]", i))
		if err != nil {
			return err
		}
		if _, err := requiredString(config, "manufacturer"); err != nil {
			return fmt.Errorf("configData[%d]: %w", i, err)
		}
		if _, err := requiredString(config, "model"); err != nil {
			return fmt.Errorf("configData[%d]: %w", i, err)
		}
	}
	return nil
}

func validateDuration(obj object) error {
	if _, err := requiredEnum(obj, "units"); err != nil {
		return err
	}
	if _, err := requiredNumber(obj, "value"); err != nil {
		return err
	}
	return nil
}

func decodeObject(data []byte) (object, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var obj object
	if err := decoder.Decode(&obj); err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.New("sapient message must be a JSON object")
	}
	return obj, nil
}

func decodeRawObject(raw json.RawMessage, field string) (object, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%s is required", field)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var obj object
	if err := decoder.Decode(&obj); err != nil {
		return nil, fmt.Errorf("%s must be an object: %w", field, err)
	}
	if obj == nil {
		return nil, fmt.Errorf("%s must be an object", field)
	}
	return obj, nil
}

func decodeRawArray(raw json.RawMessage, field string) ([]json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%s is required", field)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("%s must be an array: %w", field, err)
	}
	if items == nil {
		return nil, fmt.Errorf("%s must be an array", field)
	}
	return items, nil
}

func requiredArray(obj object, field string) ([]json.RawMessage, error) {
	items, err := decodeRawArray(obj[field], field)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%s requires at least one item", field)
	}
	return items, nil
}

func requiredTime(obj object, field string) (time.Time, error) {
	value, err := requiredString(obj, field)
	if err != nil {
		return time.Time{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339 timestamp: %w", field, err)
	}
	return parsed.UTC(), nil
}

func optionalTime(obj object, field string) (*time.Time, error) {
	value, err := optionalString(obj, field)
	if err != nil || value == nil {
		return nil, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, *value)
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339 timestamp: %w", field, err)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func requiredUUID(obj object, field string) (string, error) {
	value, err := requiredString(obj, field)
	if err != nil {
		return "", err
	}
	if !uuidPattern.MatchString(value) {
		return "", fmt.Errorf("%s must be a UUID", field)
	}
	return value, nil
}

func optionalUUID(obj object, field string) (*string, error) {
	value, err := optionalString(obj, field)
	if err != nil || value == nil {
		return nil, err
	}
	if !uuidPattern.MatchString(*value) {
		return nil, fmt.Errorf("%s must be a UUID", field)
	}
	return value, nil
}

func requiredULID(obj object, field string) (string, error) {
	value, err := requiredString(obj, field)
	if err != nil {
		return "", err
	}
	if !ulidPattern.MatchString(value) {
		return "", fmt.Errorf("%s must be a ULID", field)
	}
	return value, nil
}

func optionalULID(obj object, field string) (*string, error) {
	value, err := optionalString(obj, field)
	if err != nil || value == nil {
		return nil, err
	}
	if !ulidPattern.MatchString(*value) {
		return nil, fmt.Errorf("%s must be a ULID", field)
	}
	return value, nil
}

func requiredString(obj object, field string) (string, error) {
	value, err := optionalString(obj, field)
	if err != nil {
		return "", err
	}
	if value == nil || strings.TrimSpace(*value) == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return strings.TrimSpace(*value), nil
}

func optionalString(obj object, field string) (*string, error) {
	raw, ok := obj[field]
	if !ok || string(raw) == "null" {
		return nil, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be a string", field)
	}
	value = strings.TrimSpace(value)
	return &value, nil
}

func optionalStringArray(obj object, field string) ([]string, error) {
	raw, ok := obj[field]
	if !ok || string(raw) == "null" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%s must be a string array", field)
	}
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	return values, nil
}

func requiredEnum(obj object, field string) (string, error) {
	value, err := requiredString(obj, field)
	if err != nil {
		return "", err
	}
	if strings.HasSuffix(value, "_UNSPECIFIED") {
		return "", fmt.Errorf("%s must not be unspecified", field)
	}
	return value, nil
}

func optionalEnum(obj object, field string) (string, error) {
	value, err := optionalString(obj, field)
	if err != nil || value == nil {
		return "", err
	}
	if strings.HasSuffix(*value, "_UNSPECIFIED") {
		return "", fmt.Errorf("%s must not be unspecified", field)
	}
	return *value, nil
}

func requiredNumber(obj object, field string) (float64, error) {
	value, err := optionalNumber(obj, field)
	if err != nil {
		return 0, err
	}
	if value == nil {
		return 0, fmt.Errorf("%s is required", field)
	}
	return *value, nil
}

func optionalNumber(obj object, field string) (*float64, error) {
	raw, ok := obj[field]
	if !ok || string(raw) == "null" {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("%s must be numeric", field)
	}
	var parsed float64
	switch v := value.(type) {
	case json.Number:
		number, err := v.Float64()
		if err != nil {
			return nil, fmt.Errorf("%s must be numeric: %w", field, err)
		}
		parsed = number
	case string:
		number, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return nil, fmt.Errorf("%s must be numeric: %w", field, err)
		}
		parsed = number
	default:
		return nil, fmt.Errorf("%s must be numeric", field)
	}
	return &parsed, nil
}

func joinContent(values []ContentKind) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, string(value))
	}
	return strings.Join(parts, ", ")
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
