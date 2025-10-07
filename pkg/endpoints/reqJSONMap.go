package endpoints

import (
	"time"

	pb "github.com/Resolyte-official/gen_proto/pkg/governance"
	"github.com/Resolyte-official/service-gps/pkg/trip"
)

type ServiceStatusRequest struct{}
type ServiceStatusResponse struct {
	Status int `json:"status"`
}

func (r ServiceStatusResponse) GetStatus() int { return r.Status }

type StartTripRequest struct {
	GpsTrackerID string `json:"gps_tracker_id"`
}
type StartTripResponse struct {
	TripID string `json:"trip_id"`
	Status int    `json:"status"`
}

func (r StartTripResponse) GetStatus() int { return r.Status }

type EndTripRequest struct {
	GpsTrackerID string `json:"gps_tracker_id"`
}
type EndTripResponse struct {
	Status int `json:"status"`
}

func (r EndTripResponse) GetStatus() int { return r.Status }

type GetVehicleTripsRequest struct {
	Claims    *pb.Claims
	VehicleID string
	After     time.Time
	Before    time.Time
}
type GetVehicleTripsResponse struct {
	Trips  []*trip.Trip `json:"trips"`
	Status int          `json:"status"`
}

func (r GetVehicleTripsResponse) GetStatus() int { return r.Status }
