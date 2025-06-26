package gps

import (
	"context"
	"time"

	"github.com/Resolyte-official/service-gps/pkg/trip"
)

type Service interface {
	ServiceStatus(ctx context.Context) (int, error)
	StartTrip(ctx context.Context, gpsTrackerID string) (string, int, error)
	EndTrip(ctx context.Context, gpsTrackerID string) (int, error)
	GetVehicleTrips(ctx context.Context, schoolID, vehicleID string, after, before time.Time) ([]*trip.Trip, int, error)
}
