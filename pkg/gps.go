package gps

import (
	"context"
	"errors"
	"net/http"
	"time"

	pb "github.com/Resolyte-official/gen_proto/pkg/governance"
	"github.com/Resolyte-official/service-gps/pkg/config"
	mongodb "github.com/Resolyte-official/service-gps/pkg/db/mongo"
	svcErrors "github.com/Resolyte-official/service-gps/pkg/errors"
	"github.com/Resolyte-official/service-gps/pkg/trip"
	"github.com/Resolyte-official/service-gps/pkg/utils"
	uuidUtils "github.com/Resolyte-official/service-gps/pkg/utils/uuid"
	"github.com/google/uuid"
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
	tid, err := s.dbTripStorage.CreateTrip(ctx, vcl.SchoolID, vcl.ID, gpsTrackerID)
	if err != nil {
		return fail(err)
	}

	// convert binary bytes to uuid
	bid, ok := tid.(bson.Binary)
	if !ok {
		return fail(svcErrors.ErrInternalServer)
	}
	id, err := uuid.FromBytes(bid.Data)
	if err != nil {
		return fail(svcErrors.ErrInternalServer.WithInternalErr(err))
	}

	tripID := id.String()

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
		return fail(svcErrors.ErrTripNotFound.WithInternalErr(errors.New("no active trip was found for that gps tracker")))
	}

	// end trip stored in mongo
	if err := s.dbTripStorage.EndTrip(ctx, dt.TripID); err != nil {
		return fail(err)
	}

	return http.StatusOK, nil
}

func (s *gpsService) GetVehicleTrips(ctx context.Context, claims *pb.Claims, vehicleID string, after, before time.Time) ([]*trip.Trip, int, error) {
	fail := func(err error) ([]*trip.Trip, int, error) {
		return nil, 0, err
	}

	// validate vehicle ID
	oid, err := bson.ObjectIDFromHex(vehicleID)
	if err != nil {
		return fail(svcErrors.ErrBadRequest.WithInternalErr(errors.New("invalid vehicle_id")))
	}
	coll := s.mongoSvc.GetCollection("metadata", "vehicles")
	filter := bson.M{"_id": oid}
	res, err := s.mongoSvc.FindOne(ctx, coll, filter)
	if err != nil {
		return fail(err)
	}

	var vcl mongodb.Vehicle
	if err := res.Decode(&vcl); err != nil {
		return fail(svcErrors.ErrInternalServer.WithInternalErr(err))
	}

	id := claims.Sub
	userID, err := uuid.Parse(id)
	if err != nil {
		return fail(svcErrors.ErrInternalServer.WithInternalErr(err))
	}
	coll = s.mongoSvc.GetCollection("metadata", "users")
	filter = bson.M{"_id": uuidUtils.UUIDToBinary(userID)}
	res2, err := s.mongoSvc.FindOne(ctx, coll, filter)
	if err != nil {
		return fail(err)
	}
	var userDoc bson.M
	if err := res2.Decode(&userDoc); err != nil {
		return fail(svcErrors.ErrMongo.WithInternalErr(err))
	}
	schoolID, ok := userDoc["school_id"].(string)
	if !ok || schoolID == "" {
		return fail(svcErrors.ErrMongo.WithInternalErr(errors.New("missing metadata")))
	}

	// validate school id
	if schoolID != vcl.SchoolID {
		return fail(svcErrors.ErrAccessDenied.WithInternalErr(errors.New("vehicle is not associated with this school")))
	}

	// fetch trips b/w after and before
	trips, err := s.dbTripStorage.GetVehicleTripsFromS3(ctx, schoolID, vehicleID, &after, &before)
	if err != nil {
		return fail(err)
	}

	if len(trips) == 0 {
		return fail(svcErrors.ErrNotFound)
	}

	return trips, http.StatusOK, nil
}
