package mongo

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type CameraStream struct {
	Label    string `bson:"label" json:"label"`
	StreamID string `bson:"stream_id" json:"stream_id"`
}

type Vehicle struct {
	ID            bson.ObjectID   `bson:"_id" json:"_id"`
	VehicleType   string          `bson:"vehicle_type" json:"vehicle_type"`
	LicensePlate  string          `bson:"license_plate" json:"license_plate"`
	SchoolID      string          `bson:"school_id" json:"school_id"`
	DriverID      string          `bson:"driver_id" json:"driver_id"`
	CreatedAt     time.Time       `bson:"created_at" json:"created_at"`
	LastUpdated   time.Time       `bson:"last_updated" json:"last_updated"`
	GPSTrackerID  string          `bson:"gps_tracker_id" json:"gps_tracker_id"`
	CameraStreams []CameraStream  `bson:"camera_streams" json:"camera_streams"`
	RouteIDs      []bson.ObjectID `bson:"route_ids" json:"route_ids"`
}
