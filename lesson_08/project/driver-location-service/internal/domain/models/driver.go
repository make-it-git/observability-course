package models

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

type Driver struct {
	ID       string   `json:"id"`
	Location Location `json:"location"`
	Status   string   `json:"status"`
}
