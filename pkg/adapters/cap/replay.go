package cap

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type RawAlertRecord struct {
	Ref        string
	Source     string
	ReceivedAt time.Time
	Identifier string
	MsgType    string
	SentAt     time.Time
	RawXML     []byte
}

func (r RawAlertRecord) Alert() (Alert, error) {
	return Parse(r.RawXML)
}

type ReplayStore struct {
	path string
	mu   sync.Mutex
}

func NewReplayStore(path string) *ReplayStore {
	return &ReplayStore{path: path}
}

func (s *ReplayStore) Append(record RawAlertRecord) error {
	if s == nil || s.path == "" {
		return fmt.Errorf("cap replay store has no path")
	}
	if record.Ref == "" {
		return fmt.Errorf("cap replay record has no ref")
	}
	if len(record.RawXML) == 0 {
		return fmt.Errorf("cap replay record %q has no raw XML", record.Ref)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Dir(s.path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create replay directory: %w", err)
		}
	}
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open replay file: %w", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(record); err != nil {
		return fmt.Errorf("write replay record %q: %w", record.Ref, err)
	}
	return nil
}

func LoadReplay(path string) ([]RawAlertRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open replay file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(bufio.NewReader(file))
	records := make([]RawAlertRecord, 0)
	for {
		var record RawAlertRecord
		if err := decoder.Decode(&record); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode replay record %d: %w", len(records)+1, err)
		}
		if record.Ref == "" {
			return nil, fmt.Errorf("decode replay record %d: missing ref", len(records)+1)
		}
		if len(record.RawXML) == 0 {
			return nil, fmt.Errorf("decode replay record %q: missing raw XML", record.Ref)
		}
		records = append(records, cloneRawAlertRecord(record))
	}
	return records, nil
}

func LifecycleFixtureRecords(start time.Time) ([]RawAlertRecord, error) {
	if start.IsZero() {
		start = time.Now().UTC()
	}
	start = start.UTC()
	steps := []capLifecycleStep{
		{
			refSuffix:   "0001-alert",
			identifier:  "nws-demo-flood-warning",
			sent:        start,
			msgType:     "Alert",
			status:      "Actual",
			event:       "Flood Warning",
			urgency:     "Immediate",
			severity:    "Severe",
			certainty:   "Likely",
			effective:   start,
			expires:     start.Add(2 * time.Hour),
			headline:    "Flood Warning issued for North Branch",
			description: "Flooding is occurring near low crossings.",
			instruction: "Move to higher ground. Avoid flooded roadways.",
			areaDesc:    "North Branch",
		},
		{
			refSuffix:   "0002-update",
			identifier:  "nws-demo-flood-warning",
			sent:        start.Add(15 * time.Minute),
			msgType:     "Update",
			status:      "Actual",
			event:       "Flood Warning",
			urgency:     "Immediate",
			severity:    "Extreme",
			certainty:   "Observed",
			effective:   start.Add(15 * time.Minute),
			expires:     start.Add(3 * time.Hour),
			headline:    "Flood Warning upgraded for North Branch",
			description: "Flooding is expanding across low crossings.",
			instruction: "Evacuate low-lying areas and avoid flooded roadways.",
			areaDesc:    "North Branch",
			references:  "w-nws.webmaster@noaa.gov,nws-demo-flood-warning," + start.Format(time.RFC3339),
		},
		{
			refSuffix:   "0003-cancel",
			identifier:  "nws-demo-flood-warning",
			sent:        start.Add(45 * time.Minute),
			msgType:     "Cancel",
			status:      "Actual",
			event:       "Flood Warning",
			urgency:     "Past",
			severity:    "Severe",
			certainty:   "Observed",
			effective:   start.Add(45 * time.Minute),
			expires:     start.Add(3 * time.Hour),
			headline:    "Flood Warning cancelled for North Branch",
			description: "Flood waters are receding below warning thresholds.",
			instruction: "Continue to avoid closed low-water crossings.",
			areaDesc:    "North Branch",
			references:  "w-nws.webmaster@noaa.gov,nws-demo-flood-warning," + start.Add(15*time.Minute).Format(time.RFC3339),
		},
		{
			refSuffix:   "0004-expired",
			identifier:  "nws-demo-flood-expired",
			sent:        start.Add(-3 * time.Hour),
			msgType:     "Alert",
			status:      "Actual",
			event:       "Flood Advisory",
			urgency:     "Expected",
			severity:    "Minor",
			certainty:   "Likely",
			effective:   start.Add(-3 * time.Hour),
			expires:     start.Add(-1 * time.Hour),
			headline:    "Expired flood advisory for South Branch",
			description: "Minor flooding was previously expected near drainage channels.",
			instruction: "Monitor local conditions.",
			areaDesc:    "South Branch",
		},
	}

	records := make([]RawAlertRecord, 0, len(steps))
	for _, step := range steps {
		raw := []byte(step.xml())
		alert, err := Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("parse lifecycle fixture %s: %w", step.refSuffix, err)
		}
		records = append(records, RawAlertRecord{
			Ref:        "cap://fixture/hadr-flood/" + step.refSuffix,
			Source:     "cap:fixture:hadr-flood",
			ReceivedAt: step.sent,
			Identifier: alert.Identifier,
			MsgType:    alert.MsgType,
			SentAt:     alert.Sent,
			RawXML:     append([]byte(nil), raw...),
		})
	}
	return records, nil
}

func cloneRawAlertRecord(record RawAlertRecord) RawAlertRecord {
	record.RawXML = append([]byte(nil), record.RawXML...)
	return record
}

type capLifecycleStep struct {
	refSuffix   string
	identifier  string
	sent        time.Time
	msgType     string
	status      string
	event       string
	urgency     string
	severity    string
	certainty   string
	effective   time.Time
	expires     time.Time
	headline    string
	description string
	instruction string
	areaDesc    string
	references  string
}

func (s capLifecycleStep) xml() string {
	references := ""
	if strings.TrimSpace(s.references) != "" {
		references = "\n  <references>" + s.references + "</references>"
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<alert xmlns="%s">
  <identifier>%s</identifier>
  <sender>w-nws.webmaster@noaa.gov</sender>
  <sent>%s</sent>
  <status>%s</status>
  <msgType>%s</msgType>
  <source>NWS</source>
  <scope>Public</scope>%s
  <info>
    <language>en-US</language>
    <category>Met</category>
    <event>%s</event>
    <urgency>%s</urgency>
    <severity>%s</severity>
    <certainty>%s</certainty>
    <effective>%s</effective>
    <expires>%s</expires>
    <senderName>National Weather Service</senderName>
    <headline>%s</headline>
    <description>%s</description>
    <instruction>%s</instruction>
    <area>
      <areaDesc>%s</areaDesc>
      <polygon>38.895,-77.012 38.907,-77.011 38.908,-76.992 38.896,-76.991</polygon>
      <circle>38.900,-77.010 7.5</circle>
      <geocode>
        <valueName>SAME</valueName>
        <value>011001</value>
      </geocode>
    </area>
  </info>
</alert>`,
		NamespaceCAP12,
		s.identifier,
		s.sent.UTC().Format(time.RFC3339),
		s.status,
		s.msgType,
		references,
		s.event,
		s.urgency,
		s.severity,
		s.certainty,
		s.effective.UTC().Format(time.RFC3339),
		s.expires.UTC().Format(time.RFC3339),
		s.headline,
		s.description,
		s.instruction,
		s.areaDesc,
	)
}
