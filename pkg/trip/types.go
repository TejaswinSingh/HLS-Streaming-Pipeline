package trip

import (
	"os"
	"time"
)

type Coordinate struct {
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Timestamp time.Time `json:"timestamp"`
}

type Trip struct {
	// TripID      string       `json:"trip_id"`
	StartTime   time.Time    `json:"start_time"`
	EndTime     time.Time    `json:"end_time"`
	Coordinates []Coordinate `json:"coordinates"`
}

const (
	RWX_R_R  os.FileMode = 0o744
	RW_R_R   os.FileMode = 0o644
	YYYYMMDD string      = "2006-01-02"
)
