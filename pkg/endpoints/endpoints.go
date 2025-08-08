package endpoints

import (
	"context"

	gps "github.com/Resolyte-official/service-gps/pkg"
	"github.com/go-kit/kit/endpoint"
)

type Set struct {
	// http endpoints
	ServiceStatusEndpoint   endpoint.Endpoint
	StartTripEndpoint       endpoint.Endpoint
	EndTripEndpoint         endpoint.Endpoint
	GetVehicleTripsEndpoint endpoint.Endpoint
}

func NewEndpointSet(svc gps.Service) Set {
	return Set{
		ServiceStatusEndpoint:   MakeServiceStatusEndpoint(svc),
		StartTripEndpoint:       MakeStartTripEndpoint(svc),
		EndTripEndpoint:         MakeEndTripEndpoint(svc),
		GetVehicleTripsEndpoint: MakeGetVehicleTripsEndpoint(svc),
	}
}

func MakeServiceStatusEndpoint(svc gps.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		_ = request.(ServiceStatusRequest)
		status, err := svc.ServiceStatus(ctx)
		if err != nil {
			return nil, err
		}
		return ServiceStatusResponse{Status: status}, nil
	}
}

func MakeStartTripEndpoint(svc gps.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		r := request.(StartTripRequest)
		tripID, status, err := svc.StartTrip(ctx, r.GpsTrackerID)
		if err != nil {
			return nil, err
		}
		return StartTripResponse{
			TripID: tripID,
			Status: status,
		}, nil
	}
}

func MakeEndTripEndpoint(svc gps.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		r := request.(EndTripRequest)
		status, err := svc.EndTrip(ctx, r.GpsTrackerID)
		if err != nil {
			return nil, err
		}
		return EndTripResponse{Status: status}, nil
	}
}

func MakeGetVehicleTripsEndpoint(svc gps.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		r := request.(GetVehicleTripsRequest)
		trips, status, err := svc.GetVehicleTrips(ctx, r.Claims, r.VehicleID, r.After, r.Before)
		if err != nil {
			return nil, err
		}
		return GetVehicleTripsResponse{
			Trips:  trips,
			Status: status,
		}, nil
	}
}
