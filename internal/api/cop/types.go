package cop

import "time"

type Snapshot struct {
	GeneratedAt time.Time    `json:"generated_at"`
	Scenario    string       `json:"scenario"`
	Summary     Summary      `json:"summary"`
	Feeds       []FeedHealth `json:"feeds"`
	Assets      []Asset      `json:"assets"`
	Tracks      []Track      `json:"tracks"`
	Tasks       []Task       `json:"tasks"`
	Advisories  []Advisory   `json:"advisories"`
	Hazards     []Hazard     `json:"hazards"`
	Alerts      []Alert      `json:"alerts"`
}

type Summary struct {
	ActiveTracks     int `json:"active_tracks"`
	ActiveTasks      int `json:"active_tasks"`
	ActiveAdvisories int `json:"active_advisories"`
	ActiveAlerts     int `json:"active_alerts"`
	StaleFeeds       int `json:"stale_feeds"`
}

type FeedHealth struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	LastEventAt time.Time `json:"last_event_at"`
	Message     string    `json:"message"`
}

type Asset struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Kind       string     `json:"kind"`
	Source     string     `json:"source"`
	Position   *GeoPoint  `json:"position,omitempty"`
	Confidence float64    `json:"confidence"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Provenance Provenance `json:"provenance"`
}

type Track struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Source     string     `json:"source"`
	Status     string     `json:"status"`
	Position   GeoPoint   `json:"position"`
	Velocity   string     `json:"velocity"`
	Confidence float64    `json:"confidence"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Provenance Provenance `json:"provenance"`
}

type Task struct {
	ID          string     `json:"id"`
	Label       string     `json:"label"`
	Kind        string     `json:"kind"`
	Source      string     `json:"source"`
	Status      string     `json:"status"`
	Position    *GeoPoint  `json:"position,omitempty"`
	Description string     `json:"description,omitempty"`
	Confidence  float64    `json:"confidence"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Provenance  Provenance `json:"provenance"`
}

type Advisory struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Kind       string     `json:"kind"`
	Source     string     `json:"source"`
	Status     string     `json:"status"`
	Text       string     `json:"text"`
	Sender     string     `json:"sender,omitempty"`
	Position   *GeoPoint  `json:"position,omitempty"`
	Confidence float64    `json:"confidence"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Provenance Provenance `json:"provenance"`
}

type Hazard struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Kind       string     `json:"kind"`
	Severity   string     `json:"severity"`
	Geometry   []GeoPoint `json:"geometry"`
	Source     string     `json:"source"`
	Confidence float64    `json:"confidence"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Provenance Provenance `json:"provenance"`
}

type Alert struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Severity  string    `json:"severity"`
	Status    string    `json:"status"`
	EntityID  string    `json:"entity_id"`
	Reason    string    `json:"reason"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Provenance struct {
	Owner     string    `json:"owner"`
	SourceRef string    `json:"source_ref"`
	Observed  time.Time `json:"observed_at"`
}
