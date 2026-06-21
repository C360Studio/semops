package sapient

// TaskAckFixtureJSON returns a representative BSI Flex 335 v2-shaped task
// acknowledgement fixture used by local preflight and smoke harnesses.
func TaskAckFixtureJSON() []byte {
	return []byte(taskAckFixtureJSON)
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
