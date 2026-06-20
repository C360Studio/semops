package sapient

import (
	"strings"
	"testing"
	"time"
)

func TestParseJSONMessageAcceptsRepresentativeDstlHarnessShapes(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		content ContentKind
		assert  func(t *testing.T, msg Message)
	}{
		{
			name:    "detection report",
			body:    sampleDetectionReport,
			content: ContentDetectionReport,
			assert: func(t *testing.T, msg Message) {
				report := msg.DetectionReport
				if report == nil {
					t.Fatal("missing detection report")
				}
				if report.ReportID != "01GGYFBAXGDG7AGAHRZ6XSNY12" ||
					report.ObjectID != "01GGYFBAXH4VYRQYEX7S3XGK3H" ||
					report.TaskID == nil ||
					*report.TaskID != "01GGYFBAXHNV9DN0N74DFX2952" {
					t.Fatalf("detection ids = %+v", report)
				}
				if report.Location == nil ||
					report.Location.CoordinateSystem != "LOCATION_COORDINATE_SYSTEM_UTM_M" ||
					report.Location.Datum != "LOCATION_DATUM_WGS84_E" {
					t.Fatalf("detection location = %+v", report.Location)
				}
				if report.DetectionConfidence == nil || *report.DetectionConfidence != 0.99 {
					t.Fatalf("detection confidence = %v", report.DetectionConfidence)
				}
				if len(report.Classifications) != 1 ||
					report.Classifications[0].Type != "Human" ||
					len(report.Classifications[0].SubClass) != 1 ||
					report.Classifications[0].SubClass[0].Level != 1 {
					t.Fatalf("classification = %+v", report.Classifications)
				}
				if len(report.AssociatedDetection) != 1 ||
					report.AssociatedDetection[0].AssociationType != "ASSOCIATION_RELATION_PARENT" {
					t.Fatalf("associated detections = %+v", report.AssociatedDetection)
				}
				if !report.HasENUVelocity {
					t.Fatalf("expected ENU velocity marker")
				}
			},
		},
		{
			name:    "registration",
			body:    sampleRegistration,
			content: ContentRegistration,
			assert: func(t *testing.T, msg Message) {
				registration := msg.Registration
				if registration == nil {
					t.Fatal("missing registration")
				}
				if registration.ICDVersion != StandardVersion {
					t.Fatalf("icd version = %q", registration.ICDVersion)
				}
				if len(registration.NodeTypes) != 2 ||
					registration.NodeTypes[0] != "NODE_TYPE_CAMERA" ||
					registration.NodeTypes[1] != "NODE_TYPE_LIDAR" {
					t.Fatalf("node types = %+v", registration.NodeTypes)
				}
				if len(registration.Capabilities) != 1 ||
					registration.Capabilities[0].Category != "Test" ||
					registration.Capabilities[0].Type != "NODE_TYPE_CAMERA" {
					t.Fatalf("capabilities = %+v", registration.Capabilities)
				}
			},
		},
		{
			name:    "status report",
			body:    sampleStatusReport,
			content: ContentStatusReport,
			assert: func(t *testing.T, msg Message) {
				status := msg.StatusReport
				if status == nil {
					t.Fatal("missing status report")
				}
				if status.ReportID != "01GGYFBAV4EEPB1398MKQ47E6E" ||
					status.System != "SYSTEM_OK" ||
					status.Info != "INFO_NEW" ||
					status.Mode != "TestMode" {
					t.Fatalf("status report = %+v", status)
				}
				if status.NodeLocation == nil || status.NodeLocation.UTMZone != "30U" {
					t.Fatalf("node location = %+v", status.NodeLocation)
				}
			},
		},
		{
			name:    "task ack",
			body:    sampleTaskAck,
			content: ContentTaskAck,
			assert: func(t *testing.T, msg Message) {
				taskAck := msg.TaskAck
				if taskAck == nil {
					t.Fatal("missing task ack")
				}
				if taskAck.TaskID != "01H4R63D7NVN8444Z5M77WEBY8" ||
					taskAck.Status != "TASK_STATUS_ACCEPTED" ||
					len(taskAck.Reason) != 1 {
					t.Fatalf("task ack = %+v", taskAck)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseJSONMessage([]byte(tt.body))
			if err != nil {
				t.Fatalf("parse SAPIENT JSON: %v", err)
			}
			if msg.Content != tt.content {
				t.Fatalf("content = %q, want %q", msg.Content, tt.content)
			}
			if msg.Timestamp != time.Date(2023, 7, 7, 12, 44, 17, 27638700, time.UTC) &&
				msg.Timestamp != time.Date(2022, 11, 3, 10, 5, 6, 141204500, time.UTC) {
				t.Fatalf("unexpected timestamp = %s", msg.Timestamp)
			}
			if msg.NodeID == "" {
				t.Fatal("missing node id")
			}
			tt.assert(t, msg)
		})
	}
}

func TestParseJSONMessageRejectsMalformedFixturesBeforeProjection(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "invalid top-level uuid",
			body: strings.Replace(sampleDetectionReport, "a8654cdf-4328-47de-81fa-c495589e30c8", "not-a-uuid", 1),
			want: "nodeId",
		},
		{
			name: "missing content",
			body: `{"timestamp":"2023-07-07T12:44:17Z","nodeId":"a8654cdf-4328-47de-81fa-c495589e30c8"}`,
			want: "exactly one content",
		},
		{
			name: "multiple content",
			body: strings.Replace(sampleDetectionReport, `"detectionReport": {`, extraStatusContent+`,"detectionReport": {`, 1),
			want: "multiple content",
		},
		{
			name: "detection report id missing",
			body: strings.Replace(sampleDetectionReport, `    "reportId": "01GGYFBAXGDG7AGAHRZ6XSNY12",`+"\n", "", 1),
			want: "reportId",
		},
		{
			name: "detection location missing",
			body: strings.Replace(sampleDetectionReport, `    "location": {
      "x": 51.1739726374,
      "y": -1.82237671048,
      "z": 788,
      "xError": 5,
      "yError": 5,
      "zError": 5,
      "coordinateSystem": "LOCATION_COORDINATE_SYSTEM_UTM_M",
      "datum": "LOCATION_DATUM_WGS84_E",
      "utmZone": "30U"
    },
`, "", 1),
			want: "location",
		},
		{
			name: "registration wrong standard version",
			body: strings.Replace(sampleRegistration, StandardVersion, "BSI Flex 335 v1.0", 1),
			want: "icdVersion",
		},
		{
			name: "status report id missing",
			body: strings.Replace(sampleStatusReport, `    "reportId": "01GGYFBAV4EEPB1398MKQ47E6E",`+"\n", "", 1),
			want: "reportId",
		},
		{
			name: "task ack id missing",
			body: strings.Replace(sampleTaskAck, `    "taskId": "01H4R63D7NVN8444Z5M77WEBY8",`+"\n", "", 1),
			want: "taskId",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseJSONMessage([]byte(tt.body))
			if err == nil {
				t.Fatal("expected parse error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

// The samples below are trimmed from public Dstl BSI Flex 335 v2 Test Harness
// message shapes. They are parser preflight fixtures, not compliance evidence.

const sampleDetectionReport = `{
  "timestamp": "2023-07-07T12:44:17.027638700Z",
  "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c8",
  "destinationId": "a8654cdf-4328-47de-81fa-c495589e30c9",
  "detectionReport": {
    "reportId": "01GGYFBAXGDG7AGAHRZ6XSNY12",
    "objectId": "01GGYFBAXH4VYRQYEX7S3XGK3H",
    "taskId": "01GGYFBAXHNV9DN0N74DFX2952",
    "state": "TestState",
    "location": {
      "x": 51.1739726374,
      "y": -1.82237671048,
      "z": 788,
      "xError": 5,
      "yError": 5,
      "zError": 5,
      "coordinateSystem": "LOCATION_COORDINATE_SYSTEM_UTM_M",
      "datum": "LOCATION_DATUM_WGS84_E",
      "utmZone": "30U"
    },
    "detectionConfidence": 0.99,
    "classification": [
      {
        "type": "Human",
        "confidence": 0.8,
        "subClass": [
          {
            "type": "Male",
            "confidence": 0.6,
            "level": 1
          }
        ]
      }
    ],
    "associatedDetection": [
      {
        "timestamp": "2023-07-07T12:44:17.027638700Z",
        "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c9",
        "objectId": "01H4R63D7NVN8444Z5M77WEBY2",
        "associationType": "ASSOCIATION_RELATION_PARENT"
      }
    ],
    "derivedDetection": [
      {
        "timestamp": "2023-07-07T12:44:17.027638700Z",
        "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c9",
        "objectId": "01H4R63D7NVN8444Z5M77WEBY2"
      }
    ],
    "enu_velocity": {
      "eastRate": 1.0,
      "northRate": 2.0,
      "upRate": 3.0
    },
    "colour": "Red",
    "id": "01H4R63D7NVN8444Z5M77WEBY3"
  }
}`

const extraStatusContent = `"statusReport": {
    "reportId": "01GGYFBAV4EEPB1398MKQ47E6E",
    "system": "SYSTEM_OK",
    "info": "INFO_NEW",
    "mode": "TestMode"
  }`

const sampleRegistration = `{
  "timestamp": "2022-11-03T10:05:06.141204500Z",
  "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c9",
  "destinationId": "a8654cdf-4328-47de-81fa-c495589e30c8",
  "registration": {
    "nodeDefinition": [
      {
        "nodeType": "NODE_TYPE_CAMERA",
        "nodeSubType": ["InfraredCamera"]
      },
      {
        "nodeType": "NODE_TYPE_LIDAR",
        "nodeSubType": ["SeeThroughWall"]
      }
    ],
    "icdVersion": "BSI Flex 335 v2.0",
    "capabilities": [
      {
        "category": "Test",
        "type": "NODE_TYPE_CAMERA",
        "value": "Test",
        "units": "Url"
      }
    ],
    "statusDefinition": {
      "statusInterval": {
        "units": "TIME_UNITS_SECONDS",
        "value": 5
      }
    },
    "modeDefinition": [
      {
        "modeName": "Default",
        "modeType": "MODE_TYPE_PERMANENT",
        "settleTime": {
          "units": "TIME_UNITS_HOURS",
          "value": "30"
        },
        "task": {
          "concurrentTasks": "0",
          "regionDefinition": {
            "regionType": ["REGION_TYPE_AREA_OF_INTEREST"]
          }
        }
      }
    ],
    "configData": [
      {
        "manufacturer": "ACME",
        "model": "Vision123"
      }
    ]
  }
}`

const sampleStatusReport = `{
  "timestamp": "2023-07-07T12:44:17.027638700Z",
  "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c8",
  "destinationId": "a8654cdf-4328-47de-81fa-c495589e30c9",
  "statusReport": {
    "reportId": "01GGYFBAV4EEPB1398MKQ47E6E",
    "system": "SYSTEM_OK",
    "info": "INFO_NEW",
    "activeTaskId": "01H4R63D7NVN8444Z5M77WEBY7",
    "mode": "TestMode",
    "nodeLocation": {
      "x": 51.1739726374,
      "y": -1.82237671048,
      "z": 788,
      "coordinateSystem": "LOCATION_COORDINATE_SYSTEM_UTM_M",
      "datum": "LOCATION_DATUM_WGS84_E",
      "utmZone": "30U"
    }
  }
}`

const sampleTaskAck = `{
  "timestamp": "2023-07-07T12:44:17.027638700Z",
  "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c8",
  "destinationId": "a8654cdf-4328-47de-81fa-c495589e30c9",
  "taskAck": {
    "taskId": "01H4R63D7NVN8444Z5M77WEBY8",
    "taskStatus": "TASK_STATUS_ACCEPTED",
    "reason": ["Task was accepted by the sensor."]
  }
}`
