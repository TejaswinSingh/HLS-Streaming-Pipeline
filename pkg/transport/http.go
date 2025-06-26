package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Resolyte-official/service-gps/pkg/config"
	"github.com/Resolyte-official/service-gps/pkg/endpoints"
	svcErrors "github.com/Resolyte-official/service-gps/pkg/errors"
	grpcClient "github.com/Resolyte-official/service-gps/pkg/grpc"
	"github.com/Resolyte-official/service-gps/pkg/logging"
	httptransport "github.com/go-kit/kit/transport/http"
)

func NewHTTPHandler(ep endpoints.Set, envConfig *config.Configuration, corsConfig *CorsOptions, govClient grpcClient.GovernanceClient, lw logging.LogWriter) http.Handler {
	m := http.NewServeMux()

	m.Handle("GET /gps/healthz", httptransport.NewServer(
		ep.ServiceStatusEndpoint,
		decodeHTTPServiceStatusRequest,
		encodeResponse,
		// the below is called immediately if error occurs in any of the above
		httptransport.ServerErrorEncoder(encodeError),
	))

	m.Handle("POST /gps/start-trip", httptransport.NewServer(
		ep.StartTripEndpoint,
		decodeHTTPStartTripRequest,
		encodeResponse,
		httptransport.ServerErrorEncoder(encodeError),
	))

	m.Handle("POST /gps/end-trip", httptransport.NewServer(
		ep.EndTripEndpoint,
		decodeHTTPEndTripRequest,
		encodeResponse,
		httptransport.ServerErrorEncoder(encodeError),
	))

	m.Handle("GET /gps/get-vehicle-trips", httptransport.NewServer(
		ep.GetVehicleTripsEndpoint,
		decodeHTTPGetVehicleTripsRequest,
		encodeResponse,
		httptransport.ServerErrorEncoder(encodeError),
	))

	logger := logging.NewContextLogger(lw, "HttpHandler")

	handler := AuthMiddleware(govClient)(m)
	handler = CorsMiddleware(corsConfig)(handler)
	handler = TracingAndLoggingMiddleware(logger, envConfig)(handler)

	return handler
}

func decodeHTTPServiceStatusRequest(_ context.Context, _ *http.Request) (any, error) {
	return endpoints.ServiceStatusRequest{}, nil
}
func decodeHTTPStartTripRequest(_ context.Context, r *http.Request) (any, error) {
	gpsID := r.URL.Query().Get("gps_tracker_id")
	if gpsID == "" {
		return nil, svcErrors.ErrBadRequest
	}
	return endpoints.StartTripRequest{GpsTrackerID: gpsID}, nil
}

func decodeHTTPEndTripRequest(_ context.Context, r *http.Request) (any, error) {
	gpsID := r.URL.Query().Get("gps_tracker_id")
	if gpsID == "" {
		return nil, svcErrors.ErrBadRequest
	}
	return endpoints.EndTripRequest{GpsTrackerID: gpsID}, nil
}

func decodeHTTPGetVehicleTripsRequest(_ context.Context, r *http.Request) (any, error) {
	var req endpoints.GetVehicleTripsRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return nil, svcErrors.ErrBadRequest
	}
	return req, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	if e, ok := response.(error); ok && e != nil {
		return e
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// write status in header
	status := http.StatusOK
	if resp, ok := response.(interface{ GetStatus() int }); ok {
		status = resp.GetStatus()
	}
	w.WriteHeader(status)

	if status == http.StatusNoContent {
		return nil
	}
	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw, _ := w.(*ResponseWriter) // custom response writer

	var serviceErr *svcErrors.ServiceError
	if errors.As(err, &serviceErr) {
		rw.Err = serviceErr.DetailedError()
		w.WriteHeader(serviceErr.HttpStatus)
		json.NewEncoder(w).Encode(serviceErr)
		return
	}

	// default to internal server error if it is unknown
	// but still log the original error
	defErr := svcErrors.ErrInternalServer
	rw.Err = fmt.Sprintf("%s [Internal Error: %s]", defErr.Error(), err.Error())
	w.WriteHeader(defErr.HttpStatus)
	json.NewEncoder(w).Encode(defErr)
}
