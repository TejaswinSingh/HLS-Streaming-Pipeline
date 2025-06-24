package trip

import (
	"time"
)

type Coordinate struct {
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Timestamp time.Time `json:"timestamp"`
}

type Trip struct {
	ID        string `bson:"_id" json:"_id"`
	SchoolID  string `bson:"school_id" json:"school_id"`
	VehicleID string `bson:"vehicle_id" json:"vehicle_id"`
	GpsID     string `bson:"gps_id" json:"gps_id"`
	// StoragePath string       `bson:"storage_path" json:"storage_path"`
	StartTime   time.Time    `bson:"start_time" json:"start_time"`
	EndTime     *time.Time   `bson:"end_time,omitempty" json:"end_time,omitempty"`
	Coordinates []Coordinate `bson:"coordinates" json:"coordinates"`
}

const (
	YYYYMMDD string = "2006-01-02"
)
