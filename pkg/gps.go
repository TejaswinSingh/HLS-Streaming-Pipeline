package gps

import (
	"context"
	"errors"
	"net/http"

	"github.com/Resolyte-official/service-gps/pkg/config"
	mongodb "github.com/Resolyte-official/service-gps/pkg/db/mongo"
	svcErrors "github.com/Resolyte-official/service-gps/pkg/errors"
	"github.com/Resolyte-official/service-gps/pkg/trip"
	"github.com/Resolyte-official/service-gps/pkg/utils"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type gpsService struct {
	envConfig     *config.Configuration
	mongoSvc      *mongodb.MongoService
	dbTripStorage trip.DBTripStorage
	fetchList     *utils.FetchList
}

func NewService(cf *config.Configuration, svc *mongodb.MongoService, ts trip.DBTripStorage, fl *utils.FetchList) Service {
	return &gpsService{
		envConfig:     cf,
		mongoSvc:      svc,
		dbTripStorage: ts,
		fetchList:     fl,
	}
}

func (s *gpsService) ServiceStatus(_ context.Context) (int, error) {
	return http.StatusOK, nil
}

func (s *gpsService) StartTrip(ctx context.Context, gpsTrackerID string) (string, int, error) {
	fail := func(err error) (string, int, error) {
		return "", 0, err
	}

	// end any existing active trip
	if dt, ok := s.fetchList.Remove(gpsTrackerID); ok {
		if err := s.dbTripStorage.EndTrip(ctx, dt.TripID); err != nil {
			return fail(err)
		}
	}

	// validate gps tracker id
	coll := s.mongoSvc.GetCollection("metadata", "vehicles")
	filter := bson.M{"gps_tracker_id": gpsTrackerID}
	res, err := s.mongoSvc.FindOne(ctx, coll, filter)
	if err != nil {
		return fail(err)
	}

	var vcl mongodb.Vehicle
	if err := res.Decode(&vcl); err != nil {
		return fail(svcErrors.ErrInternalServer.WithInternalErr(err))
	}

	// create new trip in mongodb
	id, err := s.dbTripStorage.CreateTrip(ctx, vcl.SchoolID, vcl.ID, gpsTrackerID)
	if err != nil {
		return fail(err)
	}
	oid, ok := id.(bson.ObjectID)
	if !ok {
		return fail(svcErrors.ErrInternalServer)
	}

	tripID := oid.Hex()

	// add device & trip to fetch-list, now its coordinates will be fetched
	// continuously and stored in the trip, until the trip ends
	s.fetchList.Add(utils.DeviceTrip{
		GpsTrackerID: gpsTrackerID,
		TripID:       tripID,
	})

	return tripID, http.StatusOK, nil
}

func (s *gpsService) EndTrip(ctx context.Context, gpsTrackerID string) (int, error) {
	fail := func(err error) (int, error) {
		return 0, err
	}

	// remove trip (associated with the given gps tracker) from fetchlist
	dt, ok := s.fetchList.Remove(gpsTrackerID)
	if !ok {
		return fail(svcErrors.ErrInternalServer.WithInternalErr(errors.New("no active trip was found for that gps tracker")))
	}

	// end trip stored in mongo
	if err := s.dbTripStorage.EndTrip(ctx, dt.TripID); err != nil {
		return fail(err)
	}

	return http.StatusOK, nil
}
