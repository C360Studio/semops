// Package cot implements the minimal Cursor on Target XML subset SemOps needs
// for TAK/CoT fixture and replay gates.
package cot

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	TypeAirTrack         = "a-f-A-M-F-Q"
	TypeOperatorPosition = "a-f-G-U-C"
	TypeMarker           = "u-d-p"
	TypeGeoChat          = "b-t-f"
	TypeAlert            = "b-a-o-tbl"
	DefaultHow           = "m-g"
)

type Point struct {
	Lat float64
	Lon float64
	HAE float64
	CE  float64
	LE  float64
}

type Event struct {
	UID       string
	Type      string
	How       string
	Time      time.Time
	Start     time.Time
	Stale     time.Time
	Point     *Point
	Callsign  string
	SenderUID string
	CourseDeg float64
	SpeedMPS  float64
	HasTrack  bool
	Remarks   string
	ChatText  string
}

func (e Event) Validate() error {
	if strings.TrimSpace(e.UID) == "" {
		return errors.New("cot event uid is required")
	}
	if strings.TrimSpace(e.Type) == "" {
		return errors.New("cot event type is required")
	}
	return nil
}

func Marshal(e Event) ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	now := e.Time
	if now.IsZero() {
		now = time.Now().UTC()
	}
	start := e.Start
	if start.IsZero() {
		start = now
	}
	stale := e.Stale
	if stale.IsZero() {
		stale = now.Add(2 * time.Minute)
	}
	how := e.How
	if how == "" {
		how = DefaultHow
	}
	out := xmlEvent{
		Version: "2.0",
		UID:     strings.TrimSpace(e.UID),
		Type:    strings.TrimSpace(e.Type),
		How:     how,
		Time:    formatTime(now),
		Start:   formatTime(start),
		Stale:   formatTime(stale),
	}
	if e.Point != nil {
		out.Point = &xmlPoint{
			Lat: e.Point.Lat,
			Lon: e.Point.Lon,
			HAE: e.Point.HAE,
			CE:  e.Point.CE,
			LE:  e.Point.LE,
		}
	}
	detail := xmlDetail{}
	if strings.TrimSpace(e.Callsign) != "" {
		detail.Contact = &xmlContact{Callsign: strings.TrimSpace(e.Callsign)}
	}
	if e.HasTrack {
		detail.Track = &xmlTrack{Course: e.CourseDeg, Speed: e.SpeedMPS}
	}
	if strings.TrimSpace(e.Remarks) != "" {
		detail.Remarks = strings.TrimSpace(e.Remarks)
	}
	if strings.TrimSpace(e.ChatText) != "" {
		detail.Chat = &xmlChat{Message: strings.TrimSpace(e.ChatText), SenderUID: strings.TrimSpace(e.SenderUID)}
		if detail.Remarks == "" {
			detail.Remarks = strings.TrimSpace(e.ChatText)
		}
	}
	if !detail.empty() {
		out.Detail = &detail
	}
	raw, err := xml.Marshal(out)
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), raw...), nil
}

func Unmarshal(data []byte) (Event, error) {
	var in xmlEvent
	if err := xml.Unmarshal(data, &in); err != nil {
		return Event{}, err
	}
	eventTime, err := parseTime(in.Time)
	if err != nil {
		return Event{}, fmt.Errorf("cot time: %w", err)
	}
	start, err := parseOptionalTime(in.Start)
	if err != nil {
		return Event{}, fmt.Errorf("cot start: %w", err)
	}
	stale, err := parseOptionalTime(in.Stale)
	if err != nil {
		return Event{}, fmt.Errorf("cot stale: %w", err)
	}
	out := Event{
		UID:   strings.TrimSpace(in.UID),
		Type:  strings.TrimSpace(in.Type),
		How:   strings.TrimSpace(in.How),
		Time:  eventTime,
		Start: start,
		Stale: stale,
	}
	if in.Point != nil {
		out.Point = &Point{
			Lat: in.Point.Lat,
			Lon: in.Point.Lon,
			HAE: in.Point.HAE,
			CE:  in.Point.CE,
			LE:  in.Point.LE,
		}
	}
	if in.Detail != nil {
		if in.Detail.Contact != nil {
			out.Callsign = strings.TrimSpace(in.Detail.Contact.Callsign)
		}
		if in.Detail.Track != nil {
			out.CourseDeg = in.Detail.Track.Course
			out.SpeedMPS = in.Detail.Track.Speed
			out.HasTrack = true
		}
		out.Remarks = strings.TrimSpace(in.Detail.Remarks)
		if in.Detail.Chat != nil {
			out.ChatText = strings.TrimSpace(firstNonEmpty(in.Detail.Chat.Message, in.Detail.Chat.Text))
			out.SenderUID = strings.TrimSpace(firstNonEmpty(in.Detail.Chat.SenderUID, in.Detail.Chat.SenderUIDSnake))
		}
		if out.ChatText == "" && IsGeoChatType(out.Type) {
			out.ChatText = out.Remarks
		}
	}
	if err := out.Validate(); err != nil {
		return Event{}, err
	}
	return out, nil
}

func IsAirTrackType(t string) bool {
	return strings.HasPrefix(t, "a-f-A") || strings.HasPrefix(t, "a-h-A")
}

func IsOperatorType(t string) bool {
	return strings.HasPrefix(t, "a-f-G") || strings.HasPrefix(t, "a-h-G")
}

func IsMarkerType(t string) bool {
	return strings.HasPrefix(t, "u-d-p") || strings.HasPrefix(t, "b-m-p")
}

func IsGeoChatType(t string) bool {
	return strings.HasPrefix(t, TypeGeoChat)
}

func IsAlertType(t string) bool {
	return strings.HasPrefix(t, "b-a-")
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04:05.000Z", value)
}

func parseOptionalTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	return parseTime(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type xmlEvent struct {
	XMLName xml.Name   `xml:"event"`
	Version string     `xml:"version,attr,omitempty"`
	UID     string     `xml:"uid,attr"`
	Type    string     `xml:"type,attr"`
	How     string     `xml:"how,attr,omitempty"`
	Time    string     `xml:"time,attr,omitempty"`
	Start   string     `xml:"start,attr,omitempty"`
	Stale   string     `xml:"stale,attr,omitempty"`
	Point   *xmlPoint  `xml:"point,omitempty"`
	Detail  *xmlDetail `xml:"detail,omitempty"`
}

type xmlPoint struct {
	Lat float64 `xml:"lat,attr"`
	Lon float64 `xml:"lon,attr"`
	HAE float64 `xml:"hae,attr"`
	CE  float64 `xml:"ce,attr"`
	LE  float64 `xml:"le,attr"`
}

type xmlDetail struct {
	Contact *xmlContact `xml:"contact,omitempty"`
	Track   *xmlTrack   `xml:"track,omitempty"`
	Remarks string      `xml:"remarks,omitempty"`
	Chat    *xmlChat    `xml:"__chat,omitempty"`
}

func (d xmlDetail) empty() bool {
	return d.Contact == nil && d.Track == nil && d.Remarks == "" && d.Chat == nil
}

type xmlContact struct {
	Callsign string `xml:"callsign,attr,omitempty"`
}

type xmlTrack struct {
	Course float64 `xml:"course,attr,omitempty"`
	Speed  float64 `xml:"speed,attr,omitempty"`
}

type xmlChat struct {
	Message        string `xml:"message,attr,omitempty"`
	SenderUID      string `xml:"senderUid,attr,omitempty"`
	SenderUIDSnake string `xml:"sender_uid,attr,omitempty"`
	Text           string `xml:",chardata"`
}
