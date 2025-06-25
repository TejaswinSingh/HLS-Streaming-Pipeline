package trip

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	mongodb "github.com/Resolyte-official/service-gps/pkg/db/mongo"
	svcErrors "github.com/Resolyte-official/service-gps/pkg/errors"
	storage "github.com/Resolyte-official/service-gps/pkg/storage/s3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	DB             = "gps"
	Coll           = "trips"
	S3_BUCKET_NAME = "resolyte-s3"
)

type DBTripStorage interface {
	CreateTrip(ctx context.Context, schoolID string, vehicleID bson.ObjectID, gpsTrackerID string) (any, error)
	EndTrip(ctx context.Context, tripID string) error
	GetTrip(ctx context.Context, tripID string) (*Trip, error)
	AppendCoordinate(ctx context.Context, tripID string, coord Coordinate) error
	DeleteTrip(ctx context.Context, tripID string) error
	DeleteTrips(ctx context.Context, filter bson.M) (int64, error)
	GetTrips(ctx context.Context, filter bson.M) ([]*Trip, error)
}

type MongoTripStorage struct {
	mongoSvc *mongodb.MongoService
	s3svc    storage.StorageService
}

func NewDBTripStorage(mongoSvc *mongodb.MongoService, s3svc storage.StorageService) DBTripStorage {
	return &MongoTripStorage{
		mongoSvc: mongoSvc,
		s3svc:    s3svc,
	}
}

func (s *MongoTripStorage) MakeTripPath(tm time.Time, schoolID, vehicleID, tripID string) string {
	basePath := "gps/data"
	date := tm.Format(YYYYMMDD)
	return filepath.Join(basePath, schoolID, vehicleID, date, fmt.Sprintf("%s.json", tripID))
}

func (s *MongoTripStorage) CreateTrip(ctx context.Context, schoolID string, vehicleID bson.ObjectID, gpsTrackerID string) (any, error) {
	coll := s.mongoSvc.GetCollection(DB, Coll)
	now := time.Now().UTC()

	trip := Trip{
		SchoolID:     schoolID,
		VehicleID:    vehicleID,
		GpsTrackerID: gpsTrackerID,
		StartTime:    now,
		EndTime:      time.Time{},
		Coordinates:  []Coordinate{},
	}

	res, err := s.mongoSvc.InsertOne(ctx, coll, trip)
	if err != nil {
		return nil, err
	}

	return res.InsertedID, nil
}

func (s *MongoTripStorage) EndTrip(ctx context.Context, tripID string) error {
	trip, err := s.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	trip.EndTime = now

	// upload to s3
	go func(t Trip) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		buf, err := json.Marshal(t)
		if err != nil {
			// fmt.Printf("s3 upload failed: marshal trip [%s] to JSON %s", tripID, err)
			return
		}

		// make path and upload to S3
		path := s.MakeTripPath(now, trip.SchoolID, trip.VehicleID.Hex(), trip.ID.Hex())
		err = s.s3svc.UploadFile(ctx, S3_BUCKET_NAME, path, bytes.NewReader(buf))
		if err != nil {
			// mainLogger.Warnf("s3 upload failed: upload trip [%s] to s3 %s", tripID, err)
			return
		}

	}(*trip)

	// delete trip from mongo
	if err := s.DeleteTrip(ctx, tripID); err != nil {
		return err
	}

	return nil
}

func (s *MongoTripStorage) GetTrip(ctx context.Context, tripID string) (*Trip, error) {
	coll := s.mongoSvc.GetCollection(DB, Coll)

	filter := bson.M{"_id": tripID}
	res, err := s.mongoSvc.FindOne(ctx, coll, filter)
	if err != nil {
		return nil, err
	}

	var trip Trip
	if err := res.Decode(&trip); err != nil {
		return nil, svcErrors.ErrInternalServer.WithInternalErr(err)
	}

	return &trip, nil
}

func (s *MongoTripStorage) AppendCoordinate(ctx context.Context, tripID string, coord Coordinate) error {
	coll := s.mongoSvc.GetCollection(DB, Coll)

	filter := bson.M{"_id": tripID}
	update := bson.M{
		"$push": bson.M{
			"coordinates": coord,
		},
	}

	_, err := s.mongoSvc.UpdateOne(ctx, coll, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (s *MongoTripStorage) DeleteTrip(ctx context.Context, tripID string) error {
	coll := s.mongoSvc.GetCollection(DB, Coll)

	filter := bson.M{"_id": tripID}

	_, err := s.mongoSvc.DeleteOne(ctx, coll, filter)
	if err != nil {
		return err
	}

	return nil
}

// filter can include school_id, vehicleID, before date, after date, etc.
func (s *MongoTripStorage) DeleteTrips(ctx context.Context, filter bson.M) (int64, error) {
	coll := s.mongoSvc.GetCollection(DB, Coll)

	res, err := s.mongoSvc.DeleteMany(ctx, coll, filter)
	if err != nil {
		return 0, err
	}

	return res.DeletedCount, nil
}

func (s *MongoTripStorage) GetTrips(ctx context.Context, filter bson.M) ([]*Trip, error) {
	coll := s.mongoSvc.GetCollection(DB, Coll)

	cursor, err := s.mongoSvc.Find(ctx, coll, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var trips []*Trip
	for cursor.Next(ctx) {
		var trip Trip
		if err := cursor.Decode(&trip); err != nil {
			return nil, svcErrors.ErrInternalServer.WithInternalErr(err)
		}
		trips = append(trips, &trip)
	}

	if err := cursor.Err(); err != nil {
		return nil, svcErrors.ErrMongo.WithInternalErr(err)
	}

	return trips, nil
}
