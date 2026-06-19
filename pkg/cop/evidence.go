package cop

type HazardEvidenceDocument struct {
	Identifier  string                    `json:"identifier"`
	MessageType string                    `json:"message_type"`
	Status      string                    `json:"status"`
	Event       string                    `json:"event"`
	Urgency     string                    `json:"urgency"`
	Severity    string                    `json:"severity"`
	Certainty   string                    `json:"certainty"`
	AreaDesc    string                    `json:"area_desc"`
	Sender      string                    `json:"sender"`
	SenderName  string                    `json:"sender_name"`
	Sent        string                    `json:"sent,omitempty"`
	Effective   string                    `json:"effective,omitempty"`
	Expires     string                    `json:"expires,omitempty"`
	Polygons    [][]HazardEvidencePoint   `json:"polygons,omitempty"`
	Circles     []HazardEvidenceCircle    `json:"circles,omitempty"`
	Resources   []HazardEvidenceResource  `json:"resources,omitempty"`
	Geocodes    []HazardEvidenceNameValue `json:"geocodes,omitempty"`
	Parameters  []HazardEvidenceNameValue `json:"parameters,omitempty"`
}

type HazardEvidencePoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type HazardEvidenceCircle struct {
	Center   HazardEvidencePoint `json:"center"`
	RadiusKM float64             `json:"radius_km"`
}

type HazardEvidenceResource struct {
	Description string `json:"description"`
	MimeType    string `json:"mime_type"`
	URI         string `json:"uri"`
	Digest      string `json:"digest,omitempty"`
}

type HazardEvidenceNameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
