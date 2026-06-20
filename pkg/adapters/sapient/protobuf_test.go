package sapient

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestEmbeddedProtoDescriptorSetCompilesOfficialSAPIENTV2Sources(t *testing.T) {
	descriptors, err := EmbeddedProtoDescriptorSet(context.Background())
	if err != nil {
		t.Fatalf("compile embedded SAPIENT descriptors: %v", err)
	}
	if descriptors.SapientMessage == nil {
		t.Fatal("missing SapientMessage descriptor")
	}
	if got := descriptors.SapientMessage.FullName(); got != sapientMessageFullName {
		t.Fatalf("SapientMessage descriptor = %s, want %s", got, sapientMessageFullName)
	}
}

func TestParseBinaryMessageDecodesSapientMessagePayloads(t *testing.T) {
	descriptors, err := EmbeddedProtoDescriptorSet(context.Background())
	if err != nil {
		t.Fatalf("compile embedded SAPIENT descriptors: %v", err)
	}

	tests := []struct {
		name    string
		body    string
		content ContentKind
	}{
		{name: "detection report", body: sampleDetectionReport, content: ContentDetectionReport},
		{name: "registration", body: binaryRegistrationFixture, content: ContentRegistration},
		{name: "status report", body: sampleStatusReport, content: ContentStatusReport},
		{name: "task ack", body: sampleTaskAck, content: ContentTaskAck},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := mustBinaryPayload(t, descriptors, tt.body)
			message, err := ParseBinaryMessage(payload, descriptors)
			if err != nil {
				t.Fatalf("parse binary SAPIENT message: %v", err)
			}
			if message.Content != tt.content {
				t.Fatalf("content = %q, want %q", message.Content, tt.content)
			}
			if message.NodeID == "" || message.Timestamp.IsZero() {
				t.Fatalf("envelope = %+v", message)
			}
		})
	}
}

func TestParseBinaryMessageRejectsMandatoryFieldFailures(t *testing.T) {
	descriptors, err := EmbeddedProtoDescriptorSet(context.Background())
	if err != nil {
		t.Fatalf("compile embedded SAPIENT descriptors: %v", err)
	}
	body := strings.Replace(sampleDetectionReport, `    "reportId": "01GGYFBAXGDG7AGAHRZ6XSNY12",`+"\n", "", 1)
	payload := mustBinaryPayload(t, descriptors, body)

	_, err = ParseBinaryMessage(payload, descriptors)
	if err == nil {
		t.Fatal("expected missing reportId failure")
	}
	if !strings.Contains(err.Error(), "reportId") {
		t.Fatalf("error = %v, want reportId", err)
	}
}

func mustBinaryPayload(t *testing.T, descriptors *ProtoDescriptorSet, body string) []byte {
	t.Helper()
	dynamic := dynamicpb.NewMessage(descriptors.SapientMessage)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal([]byte(body), dynamic); err != nil {
		t.Fatalf("unmarshal JSON fixture into dynamic SAPIENT message: %v", err)
	}
	payload, err := proto.Marshal(dynamic)
	if err != nil {
		t.Fatalf("marshal dynamic SAPIENT message: %v", err)
	}
	return payload
}

const binaryRegistrationFixture = `{
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
          "value": 30
        },
        "task": {
          "concurrentTasks": 0,
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
