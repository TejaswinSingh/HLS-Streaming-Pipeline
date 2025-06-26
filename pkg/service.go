package gps

import (
	"context"

	"github.com/Resolyte-official/service-gps/pkg/trip"
	"github.com/Resolyte-official/service-gps/pkg/utils/types"
)

type Service interface {
	ServiceStatus(ctx context.Context) (int, error)
	StartTrip(ctx context.Context, gpsTrackerID string) (string, int, error)
	EndTrip(ctx context.Context, gpsTrackerID string) (int, error)
	GetVehicleTrips(ctx context.Context, schoolID, vehicleID string, after, before types.Date) ([]*trip.Trip, int, error)
}
