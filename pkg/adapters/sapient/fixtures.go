package sapient

// TaskAckFixtureJSON returns a representative BSI Flex 335 v2-shaped task
// acknowledgement fixture used by local preflight and smoke harnesses.
func TaskAckFixtureJSON() []byte {
	return []byte(taskAckFixtureJSON)
}

// DetectionFixtureJSON returns a representative absolute-location detection
// report fixture for graph-projection smoke paths. It intentionally uses
// LAT_LNG_DEG_M plus WGS84 so projection can avoid UTM or range/bearing guesses.
func DetectionFixtureJSON() []byte {
	return []byte(detectionFixtureJSON)
}

const taskAckFixtureJSON = `{
  "timestamp": "2023-07-07T12:44:17.027638700Z",
  "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c8",
  "taskAck": {
    "taskId": "01H4R63D7NVN8444Z5M77WEBY8",
    "taskStatus": "TASK_STATUS_ACCEPTED",
    "reason": ["accepted for preflight"]
  }
}`

const detectionFixtureJSON = `{
  "timestamp": "2023-07-07T12:44:17.027638700Z",
  "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c8",
  "destinationId": "a8654cdf-4328-47de-81fa-c495589e30c9",
  "detectionReport": {
    "reportId": "01GGYFBAXGDG7AGAHRZ6XSNY12",
    "objectId": "01GGYFBAXH4VYRQYEX7S3XGK3H",
    "taskId": "01GGYFBAXHNV9DN0N74DFX2952",
    "state": "TestState",
    "location": {
      "x": -1.82237671048,
      "y": 51.1739726374,
      "z": 788,
      "coordinateSystem": "LOCATION_COORDINATE_SYSTEM_LAT_LNG_DEG_M",
      "datum": "LOCATION_DATUM_WGS84_E"
    },
    "detectionConfidence": 0.91
  }
}`
