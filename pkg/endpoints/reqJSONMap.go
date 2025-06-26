package endpoints

import (
	"github.com/Resolyte-official/service-gps/pkg/trip"
	"github.com/Resolyte-official/service-gps/pkg/utils/types"
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
	SchoolID  string     `json:"school_id"`
	VehicleID string     `json:"vehicle_id"`
	After     types.Date `json:"after"`
	Before    types.Date `json:"before"`
}
type GetVehicleTripsResponse struct {
	Trips  []*trip.Trip `json:"trips"`
	Status int          `json:"status"`
}

func (r GetVehicleTripsResponse) GetStatus() int { return r.Status }
