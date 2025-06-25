package endpoints

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
