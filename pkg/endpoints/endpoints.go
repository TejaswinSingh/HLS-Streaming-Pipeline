package endpoints

import (
	"context"

	gps "github.com/Resolyte-official/service-gps/pkg"
	"github.com/go-kit/kit/endpoint"
)

type Set struct {
	// http endpoints
	ServiceStatusEndpoint endpoint.Endpoint
}

func NewEndpointSet(svc gps.Service) Set {
	return Set{
		ServiceStatusEndpoint: MakeServiceStatusEndpoint(svc),
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
