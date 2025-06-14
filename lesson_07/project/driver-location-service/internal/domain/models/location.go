package models

type GpsPoint struct {
	DriverID  string   `json:"driver_id"`
	Location  Location `json:"location"`
	Timestamp int64    `json:"timestamp"`
	Speed     float64  `json:"speed"`
}
