package trip

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Coordinate struct {
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Timestamp time.Time `json:"timestamp"`
}

type Trip struct {
	ID           bson.Binary   `bson:"_id" json:"_id"`
	SchoolID     string        `bson:"school_id" json:"school_id"`
	VehicleID    bson.ObjectID `bson:"vehicle_id" json:"vehicle_id"`
	GpsTrackerID string        `bson:"gps_tracker_id" json:"gps_tracker_id"`
	StartTime    time.Time     `bson:"start_time" json:"start_time"`
	EndTime      time.Time     `bson:"end_time,omitempty" json:"end_time,omitempty"`
	Coordinates  []Coordinate  `bson:"coordinates" json:"coordinates"`
}

const (
	YYYYMMDD string = "2006-01-02"
)
