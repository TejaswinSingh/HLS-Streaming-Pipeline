package trip

import (
	"context"
	"time"

	mongodb "github.com/Resolyte-official/service-gps/pkg/db/mongo"
	svcErrors "github.com/Resolyte-official/service-gps/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	DB   = "gps"
	Coll = "trips"
)

type DBTripStorage interface {
	CreateTrip(ctx context.Context, schoolID, busID, gpsID, tripID string) error
	EndTrip(ctx context.Context, tripID string) error
	GetTrip(ctx context.Context, tripID string) (*Trip, error)
	AppendCoordinate(ctx context.Context, tripID string, coord Coordinate) error
	DeleteTrip(ctx context.Context, tripID string) error
	DeleteTrips(ctx context.Context, filter bson.M) (int64, error)
	GetTrips(ctx context.Context, filter bson.M) ([]*Trip, error)
}

type MongoTripStorage struct {
	mongoSvc *mongodb.MongoService
}

func NewDBTripStorage(mongoSvc *mongodb.MongoService) DBTripStorage {
	return &MongoTripStorage{
		mongoSvc: mongoSvc,
	}
}

func (s *MongoTripStorage) CreateTrip(ctx context.Context, schoolID, busID, gpsID, tripID string) error {
	coll := s.mongoSvc.GetCollection(DB, Coll)
	now := time.Now().UTC()

	trip := Trip{
		ID:          tripID,
		SchoolID:    schoolID,
		BusID:       busID,
		GpsID:       gpsID,
		StartTime:   now,
		EndTime:     nil,
		Coordinates: []Coordinate{},
	}

	_, err := s.mongoSvc.InsertOne(ctx, coll, trip)
	if err != nil {
		return err
	}

	return nil
}

func (s *MongoTripStorage) EndTrip(ctx context.Context, tripID string) error {
	coll := s.mongoSvc.GetCollection(DB, Coll)

	filter := bson.M{"_id": tripID}
	update := bson.M{
		"$set": bson.M{
			"end_time": time.Now().UTC(),
		},
	}

	_, err := s.mongoSvc.UpdateOne(ctx, coll, filter, update)
	if err != nil {
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

// filter can include school_id, bus_id, before date, after date, etc.
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
