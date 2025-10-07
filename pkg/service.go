package gps

import (
	"context"
	"time"

	pb "github.com/Resolyte-official/gen_proto/pkg/governance"
	"github.com/Resolyte-official/service-gps/pkg/trip"
)

type Service interface {
	ServiceStatus(ctx context.Context) (int, error)
	StartTrip(ctx context.Context, gpsTrackerID string) (string, int, error)
	EndTrip(ctx context.Context, gpsTrackerID string) (int, error)
	GetVehicleTrips(ctx context.Context, claims *pb.Claims, vehicleID string, after, before time.Time) ([]*trip.Trip, int, error)
}
