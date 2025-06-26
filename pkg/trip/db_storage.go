package trip

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	mongodb "github.com/Resolyte-official/service-gps/pkg/db/mongo"
	svcErrors "github.com/Resolyte-official/service-gps/pkg/errors"
	storage "github.com/Resolyte-official/service-gps/pkg/storage/s3"
	uuidUtils "github.com/Resolyte-official/service-gps/pkg/utils/uuid"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	DB             = "gps"
	Coll           = "trips"
	S3_BUCKET_NAME = "resolyte-s3"
	BASE_PATH      = "gps/data"
)

type DBTripStorage interface {
	CreateTrip(ctx context.Context, schoolID string, vehicleID bson.ObjectID, gpsTrackerID string) (any, error)
	EndTrip(ctx context.Context, tripID string) error
	GetTrip(ctx context.Context, tripID string) (*Trip, error)
	AppendCoordinate(ctx context.Context, tripID string, coord Coordinate) error
	DeleteTrip(ctx context.Context, tripID string) error
	GetVehicleTripsFromS3(ctx context.Context, schoolID string, vehicleID string, after, before *time.Time) ([]*Trip, error)
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
	date := tm.Format(YYYYMMDD)
	return filepath.Join(BASE_PATH, schoolID, vehicleID, date, fmt.Sprintf("%s.json", tripID))
}

func (s *MongoTripStorage) CreateTrip(ctx context.Context, schoolID string, vehicleID bson.ObjectID, gpsTrackerID string) (any, error) {
	coll := s.mongoSvc.GetCollection(DB, Coll)
	now := time.Now().UTC()

	id := uuidUtils.NewUUIDv4()
	bid := bson.Binary{
		Subtype: 0x04,
		Data:    id[:],
	}

	trip := Trip{
		ID:           bid,
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
			fmt.Printf("s3 upload failed: marshal trip [%s] to JSON %s\n", tripID, err)
			return
		}

		uuidVal, err := uuid.FromBytes(trip.ID.Data)
		if err != nil {
			fmt.Printf("s3 upload failed: extract uuid from tripID %s\n", err)
			return
		}

		// make path and upload to S3
		path := s.MakeTripPath(now, trip.SchoolID, trip.VehicleID.Hex(), uuidVal.String())
		err = s.s3svc.UploadFile(ctx, S3_BUCKET_NAME, path, bytes.NewReader(buf))
		if err != nil {
			fmt.Printf("s3 upload failed: upload trip [%s] to s3 %s\n", tripID, err)
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

	id, err := uuid.Parse(tripID)
	if err != nil {
		return nil, svcErrors.ErrBadRequest.WithInternalErr(err)
	}

	filter := bson.M{
		"_id": bson.Binary{
			Subtype: 0x04,
			Data:    id[:],
		},
	}
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

	id, err := uuid.Parse(tripID)
	if err != nil {
		return svcErrors.ErrBadRequest.WithInternalErr(err)
	}

	filter := bson.M{
		"_id": bson.Binary{
			Subtype: 0x04,
			Data:    id[:],
		},
	}
	update := bson.M{
		"$push": bson.M{
			"coordinates": coord,
		},
	}

	_, err = s.mongoSvc.UpdateOne(ctx, coll, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (s *MongoTripStorage) DeleteTrip(ctx context.Context, tripID string) error {
	coll := s.mongoSvc.GetCollection(DB, Coll)

	id, err := uuid.Parse(tripID)
	if err != nil {
		return svcErrors.ErrBadRequest.WithInternalErr(err)
	}

	filter := bson.M{
		"_id": bson.Binary{
			Subtype: 0x04,
			Data:    id[:],
		},
	}

	_, err = s.mongoSvc.DeleteOne(ctx, coll, filter)
	if err != nil {
		return err
	}

	return nil
}

func (s *MongoTripStorage) GetVehicleTripsFromS3(
	ctx context.Context,
	schoolID string,
	vehicleID string,
	after, before *time.Time, // optional: nil means no time filter
) ([]*Trip, error) {
	prefix := filepath.Join(BASE_PATH, schoolID, vehicleID) + "/"
	const maxConcurrency = 10

	objectKeys, err := s.s3svc.ListFiles(ctx, S3_BUCKET_NAME, prefix)
	if err != nil {
		return nil, svcErrors.ErrInternalServer.WithInternalErr(err)
	}

	// Filter keys by date, if before/after filters are set
	filteredKeys := make([]string, 0, len(objectKeys))
	for _, key := range objectKeys {
		dateStr := extractDateFromKey(key)
		if dateStr == "" {
			continue
		}
		fileDate, err := time.Parse(YYYYMMDD, dateStr)
		if err != nil {
			continue
		}
		if after != nil && !fileDate.After(*after) {
			continue
		}
		if before != nil && !fileDate.Before(*before) {
			continue
		}
		filteredKeys = append(filteredKeys, key)
	}

	type tripResult struct {
		trip *Trip
		err  error
	}

	// Download files in parallel
	results := make(chan tripResult, len(filteredKeys))
	sem := make(chan struct{}, maxConcurrency)

	var wg sync.WaitGroup
	for _, key := range filteredKeys {
		wg.Add(1)

		go func(k string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			data, err := s.s3svc.DownloadFile(ctx, S3_BUCKET_NAME, k)
			if err != nil {
				results <- tripResult{nil, fmt.Errorf("download failed for %s: %w", k, err)}
				return
			}
			var trip Trip
			if err := json.Unmarshal(data, &trip); err != nil {
				results <- tripResult{nil, fmt.Errorf("unmarshal failed for %s: %w", k, err)}
				return
			}
			results <- tripResult{&trip, nil}
		}(key)
	}

	wg.Wait()
	close(results)

	// Collect only successfully downloaded trip files
	var trips []*Trip
	for res := range results {
		if res.err != nil {
			// fmt.Println(res.err) // TODO: log properly debug or warn
			continue
		}
		trips = append(trips, res.trip)
	}

	return trips, nil
}

func extractDateFromKey(key string) string {
	// Expected key format: gps/data/schoolID/vehicleID/YYYY-MM-DD/tripID.json
	parts := strings.Split(key, "/")
	part := parts[len(strings.Split(BASE_PATH, "/"))+2]

	if len(part) == 10 && part[4] == '-' && part[7] == '-' {
		return part
	}
	return ""
}
