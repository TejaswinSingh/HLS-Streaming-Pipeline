package gps

import (
	"context"
)

type Service interface {
	ServiceStatus(ctx context.Context) (int, error)
	StartTrip(ctx context.Context, gpsTrackerID string) (string, int, error)
	EndTrip(ctx context.Context, gpsTrackerID string) (int, error)
}
