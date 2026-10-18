package models

type GpsPoint struct {
	DriverID  string         `json:"driver_id"`
	Location  Location       `json:"location"`
	Timestamp int64          `json:"timestamp"`
	Speed     float64        `json:"speed"`
	Analysis  *PointAnalysis `json:"analysis,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type PointAnalysis struct {
	CalculatedSpeed   float64   `json:"calculated_speed"`
	EstimatedLocation *Location `json:"estimated_location,omitempty"`
	Confidence        float64   `json:"confidence"`
	IsAnomaly         bool      `json:"is_anomaly"`
}

type TrackAnalysis struct {
	DriverID     string     `json:"driver_id"`
	Points       []GpsPoint `json:"points"`
	AverageSpeed float64    `json:"average_speed"`
	MaxSpeed     float64    `json:"max_speed"`
	Confidence   float64    `json:"confidence"`
	AnomalyCount int        `json:"anomaly_count"`
	Timestamp    int64      `json:"timestamp"`
}
