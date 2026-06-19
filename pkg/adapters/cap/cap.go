// Package cap implements the CAP 1.2 XML subset SemOps needs for
// deterministic civilian-warning fixtures.
package cap

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const NamespaceCAP12 = "urn:oasis:names:tc:emergency:cap:1.2"

type Alert struct {
	Identifier string
	Sender     string
	Sent       time.Time
	Status     string
	MsgType    string
	Source     string
	Scope      string
	References string
	Infos      []Info
}

type Info struct {
	Language    string
	Categories  []string
	Event       string
	Urgency     string
	Severity    string
	Certainty   string
	Effective   time.Time
	Expires     time.Time
	SenderName  string
	Headline    string
	Description string
	Instruction string
	Web         string
	Contact     string
	Parameters  []NameValue
	Resources   []Resource
	Areas       []Area
}

type Area struct {
	AreaDesc string
	Polygons [][]Point
	Circles  []Circle
	Geocodes []NameValue
}

type Point struct {
	Lat float64
	Lon float64
}

type Circle struct {
	Center   Point
	RadiusKM float64
}

type NameValue struct {
	Name  string
	Value string
}

type Resource struct {
	Description string
	MimeType    string
	URI         string
	Digest      string
}

func Parse(data []byte) (Alert, error) {
	var raw xmlAlert
	if err := xml.Unmarshal(data, &raw); err != nil {
		return Alert{}, err
	}
	alert, err := raw.toAlert()
	if err != nil {
		return Alert{}, err
	}
	if err := alert.Validate(); err != nil {
		return Alert{}, err
	}
	return alert, nil
}

func (a Alert) Validate() error {
	if strings.TrimSpace(a.Identifier) == "" {
		return errors.New("cap alert identifier is required")
	}
	if strings.TrimSpace(a.Sender) == "" {
		return errors.New("cap alert sender is required")
	}
	if a.Sent.IsZero() {
		return errors.New("cap alert sent time is required")
	}
	if strings.TrimSpace(a.Status) == "" {
		return errors.New("cap alert status is required")
	}
	if strings.TrimSpace(a.MsgType) == "" {
		return errors.New("cap alert msgType is required")
	}
	if strings.TrimSpace(a.Scope) == "" {
		return errors.New("cap alert scope is required")
	}
	if len(a.Infos) == 0 {
		return errors.New("cap alert requires at least one info block")
	}
	return nil
}

func (a Alert) PrimaryInfo() (Info, bool) {
	if len(a.Infos) == 0 {
		return Info{}, false
	}
	return a.Infos[0], true
}

func (i Info) AdvisoryText() string {
	parts := make([]string, 0, 3)
	for _, value := range []string{i.Headline, i.Description, i.Instruction} {
		if value = strings.TrimSpace(value); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, "\n\n")
}

type xmlAlert struct {
	XMLName    xml.Name  `xml:"alert"`
	Identifier string    `xml:"identifier"`
	Sender     string    `xml:"sender"`
	Sent       string    `xml:"sent"`
	Status     string    `xml:"status"`
	MsgType    string    `xml:"msgType"`
	Source     string    `xml:"source"`
	Scope      string    `xml:"scope"`
	References string    `xml:"references"`
	Infos      []xmlInfo `xml:"info"`
}

type xmlInfo struct {
	Language    string         `xml:"language"`
	Categories  []string       `xml:"category"`
	Event       string         `xml:"event"`
	Urgency     string         `xml:"urgency"`
	Severity    string         `xml:"severity"`
	Certainty   string         `xml:"certainty"`
	Effective   string         `xml:"effective"`
	Expires     string         `xml:"expires"`
	SenderName  string         `xml:"senderName"`
	Headline    string         `xml:"headline"`
	Description string         `xml:"description"`
	Instruction string         `xml:"instruction"`
	Web         string         `xml:"web"`
	Contact     string         `xml:"contact"`
	Parameters  []xmlNameValue `xml:"parameter"`
	Resources   []xmlResource  `xml:"resource"`
	Areas       []xmlArea      `xml:"area"`
}

type xmlArea struct {
	AreaDesc string         `xml:"areaDesc"`
	Polygons []string       `xml:"polygon"`
	Circles  []string       `xml:"circle"`
	Geocodes []xmlNameValue `xml:"geocode"`
}

type xmlNameValue struct {
	Name  string `xml:"valueName"`
	Value string `xml:"value"`
}

type xmlResource struct {
	Description string `xml:"resourceDesc"`
	MimeType    string `xml:"mimeType"`
	URI         string `xml:"uri"`
	Digest      string `xml:"digest"`
}

func (x xmlAlert) toAlert() (Alert, error) {
	sent, err := parseOptionalTime(x.Sent)
	if err != nil {
		return Alert{}, fmt.Errorf("cap sent: %w", err)
	}
	out := Alert{
		Identifier: strings.TrimSpace(x.Identifier),
		Sender:     strings.TrimSpace(x.Sender),
		Sent:       sent,
		Status:     strings.TrimSpace(x.Status),
		MsgType:    strings.TrimSpace(x.MsgType),
		Source:     strings.TrimSpace(x.Source),
		Scope:      strings.TrimSpace(x.Scope),
		References: strings.TrimSpace(x.References),
		Infos:      make([]Info, 0, len(x.Infos)),
	}
	for i, raw := range x.Infos {
		info, err := raw.toInfo()
		if err != nil {
			return Alert{}, fmt.Errorf("cap info %d: %w", i+1, err)
		}
		out.Infos = append(out.Infos, info)
	}
	return out, nil
}

func (x xmlInfo) toInfo() (Info, error) {
	effective, err := parseOptionalTime(x.Effective)
	if err != nil {
		return Info{}, fmt.Errorf("effective: %w", err)
	}
	expires, err := parseOptionalTime(x.Expires)
	if err != nil {
		return Info{}, fmt.Errorf("expires: %w", err)
	}
	out := Info{
		Language:    strings.TrimSpace(x.Language),
		Categories:  trimStrings(x.Categories),
		Event:       strings.TrimSpace(x.Event),
		Urgency:     strings.TrimSpace(x.Urgency),
		Severity:    strings.TrimSpace(x.Severity),
		Certainty:   strings.TrimSpace(x.Certainty),
		Effective:   effective,
		Expires:     expires,
		SenderName:  strings.TrimSpace(x.SenderName),
		Headline:    strings.TrimSpace(x.Headline),
		Description: strings.TrimSpace(x.Description),
		Instruction: strings.TrimSpace(x.Instruction),
		Web:         strings.TrimSpace(x.Web),
		Contact:     strings.TrimSpace(x.Contact),
		Parameters:  convertNameValues(x.Parameters),
		Resources:   make([]Resource, 0, len(x.Resources)),
		Areas:       make([]Area, 0, len(x.Areas)),
	}
	for _, resource := range x.Resources {
		out.Resources = append(out.Resources, Resource{
			Description: strings.TrimSpace(resource.Description),
			MimeType:    strings.TrimSpace(resource.MimeType),
			URI:         strings.TrimSpace(resource.URI),
			Digest:      strings.TrimSpace(resource.Digest),
		})
	}
	for i, raw := range x.Areas {
		area, err := raw.toArea()
		if err != nil {
			return Info{}, fmt.Errorf("area %d: %w", i+1, err)
		}
		out.Areas = append(out.Areas, area)
	}
	return out, nil
}

func (x xmlArea) toArea() (Area, error) {
	out := Area{
		AreaDesc: strings.TrimSpace(x.AreaDesc),
		Polygons: make([][]Point, 0, len(x.Polygons)),
		Circles:  make([]Circle, 0, len(x.Circles)),
		Geocodes: convertNameValues(x.Geocodes),
	}
	for i, raw := range x.Polygons {
		polygon, err := parsePolygon(raw)
		if err != nil {
			return Area{}, fmt.Errorf("polygon %d: %w", i+1, err)
		}
		out.Polygons = append(out.Polygons, polygon)
	}
	for i, raw := range x.Circles {
		circle, err := parseCircle(raw)
		if err != nil {
			return Area{}, fmt.Errorf("circle %d: %w", i+1, err)
		}
		out.Circles = append(out.Circles, circle)
	}
	return out, nil
}

func parsePolygon(raw string) ([]Point, error) {
	pairs := strings.Fields(raw)
	if len(pairs) < 3 {
		return nil, errors.New("polygon requires at least three points")
	}
	points := make([]Point, 0, len(pairs))
	for _, pair := range pairs {
		point, err := parsePoint(pair)
		if err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}

func parseCircle(raw string) (Circle, error) {
	parts := strings.Fields(raw)
	if len(parts) != 2 {
		return Circle{}, errors.New("circle must be '<lat,lon> <radius-km>'")
	}
	center, err := parsePoint(parts[0])
	if err != nil {
		return Circle{}, err
	}
	radius, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return Circle{}, fmt.Errorf("circle radius: %w", err)
	}
	if radius <= 0 {
		return Circle{}, errors.New("circle radius must be greater than zero")
	}
	return Circle{Center: center, RadiusKM: radius}, nil
}

func parsePoint(raw string) (Point, error) {
	latText, lonText, ok := strings.Cut(strings.TrimSpace(raw), ",")
	if !ok {
		return Point{}, fmt.Errorf("point %q must be lat,lon", raw)
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(latText), 64)
	if err != nil {
		return Point{}, fmt.Errorf("point latitude: %w", err)
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(lonText), 64)
	if err != nil {
		return Point{}, fmt.Errorf("point longitude: %w", err)
	}
	if lat < -90 || lat > 90 {
		return Point{}, fmt.Errorf("point latitude %f out of range", lat)
	}
	if lon < -180 || lon > 180 {
		return Point{}, fmt.Errorf("point longitude %f out of range", lon)
	}
	return Point{Lat: lat, Lon: lon}, nil
}

func parseOptionalTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC(), nil
	}
	return time.Parse(time.RFC3339, value)
}

func trimStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func convertNameValues(values []xmlNameValue) []NameValue {
	out := make([]NameValue, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		text := strings.TrimSpace(value.Value)
		if name != "" || text != "" {
			out = append(out, NameValue{Name: name, Value: text})
		}
	}
	return out
}
